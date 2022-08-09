package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"sc.y3/console"
	//"github.com/rivo/tview"
)

func main() {
	dispatcherFlag := flag.String("dispatcher", "localhost:8080", "Dispatcher address")
	flag.Parse()

	if *dispatcherFlag == "localhost:8080" && os.Getenv("DISPATCHER") != "" {
		*dispatcherFlag = os.Getenv("DISPATCHER")
	}

	err := console.DialDispatcher(*dispatcherFlag)
	if err != nil {
		log.Panic(err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("? ")
		scanner.Scan()
		input := scanner.Text()
		command := console.Parse(input)
		result, err := command.Exec()
		if err != nil {
			fmt.Printf("> [ERROR] %s\n", err)
		} else {
			fmt.Println("> ", result)
		}
	}
}
