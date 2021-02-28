/*
Copyright Zhigui.com. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:       "import",
	Short:     "导入查询记录",
	Long:      "指定文件路径，读取文件中的交易信息，将数据导入到链上",
	ValidArgs: []string{"1"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return TransferTxFromCSV(cmd, args)
	},
}

type TransferTx struct {
	From  string `json:"from" csv:"from"`
	To    string `json:"to" csv:"to"`
	Value int64  `json:"value" csv:"value"`
	Fee   int64  `json:"fee" csv:"fee"`
}

func TransferTxFromCSV(cmd *cobra.Command, args []string) error {
	csvFile, err := os.Open(FilePath)
	if err != nil {
		return errors.Errorf("Couldn't open the csv file, err:%s", err.Error())
	}
	defer csvFile.Close()

	r := csv.NewReader(csvFile)

	var header []string
	fieldIndex := func(fieldName string) int {
		for k, v := range header {
			if v == fieldName {
				return k
			}
		}
		return -1
	}

	count := 0
	var wg sync.WaitGroup
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header == nil {
			header = record
			continue
		}

		tx := &TransferTx{}
		err = unmarshalCSV(record, fieldIndex, tx)
		if err != nil {
			logger.Error("unmarshal csv failed", "error", err.Error())
		}
		logger.Info("unmarshal csv tx data", "tx", tx)
		count++

		wg.Add(1)
		// TODO
		go sendTransferTxRequest(tx, count, &wg)

		if Interval > 0 {
			time.Sleep(time.Duration(Interval))
		}
	}

	wg.Wait()

	logger.Info("all txs write in blockchian complete!", "count", count)
	return nil
}

func sendTransferTxRequest(tx *TransferTx, count int, wg *sync.WaitGroup) {
	recordBytes, err := json.Marshal(tx)
	if err != nil {
		panic(fmt.Sprintf("Marshal auth record batch index:%d failed, %s", count, err.Error()))
	}

	resp, err := http.Post("http://45.76.98.80/invoke",
		"application/x-www-form-urlencoded",
		bytes.NewReader(recordBytes))
	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(body)
	}
	wg.Done()
}

func unmarshalCSV(record []string, fieldIndex func(fieldName string) int, ptr interface{}) error {
	t := reflect.TypeOf(ptr)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return errors.New("params must be struct ptr")
	}

	v := reflect.ValueOf(ptr).Elem()

	for i := 0; i < v.NumField(); i++ {
		tag := v.Type().Field(i).Tag.Get("csv")
		if tag == "" || tag == "-" {
			continue
		}
		k := fieldIndex(tag)
		if k < 0 {
			continue
		}

		f := v.Field(i)
		switch f.Type().String() {
		case "string":
			f.SetString(record[k])
		case "int64":
			var ival int64
			var err error
			if record[k] == "" {
				ival = 0
			} else {
				ival, err = strconv.ParseInt(record[k], 10, 64)
				if err != nil {
					return err
				}
			}
			f.SetInt(ival)
		default:
			return errors.Errorf("unsupported type [%s]", f.Type().String())
		}
	}
	return nil
}
