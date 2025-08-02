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
	log.Println(c.Sprintln("Warn: ", s))
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
