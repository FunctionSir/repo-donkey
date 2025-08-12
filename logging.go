/*
 * @Author: FunctionSir
 * @License: AGPLv3
 * @Date: 2025-07-28 20:40:51
 * @LastEditTime: 2025-08-03 21:43:50
 * @LastEditors: FunctionSir
 * @Description: -
 * @FilePath: /repo-donkey/logging.go
 */
package main

import (
	"log"

	"github.com/fatih/color"
)

func LogError(s string) {
	c := color.New(color.FgHiRed, color.Underline)
	log.Println(c.Sprintln("Error:", s))
	panic(s)
}

func LogWarn(s string) {
	c := color.New(color.FgHiYellow)
	log.Println(c.Sprint("Warn: ", s))
}

func LogInfo(s string) {
	c := color.New(color.FgHiGreen)
	log.Println(c.Sprint("Info: ", s))
}

func Check(err error) {
	if err != nil {
		LogError(err.Error())
	}
}
