package cmd

import (
	"bufio"
	"log"
	"os"
)

func ListenCmdInput() {
	rd := bufio.NewReader(os.Stdin)
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			log.Println(err)
			continue
		}
		log.Printf(line)
	}
}
