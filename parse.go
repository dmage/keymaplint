package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/dmage/keymaplint/scanner"
	"github.com/dmage/keymaplint/token"
)

func main() {
	for _, filename := range os.Args[1:] {
		f, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}

		data, err := ioutil.ReadAll(f)
		if err != nil {
			log.Fatal(err)
		}

		l := scanner.New(filename, string(data))
		sep := ""
		for {
			typ, val := l.Scan()
			if typ == token.EOF {
				break
			}
			if typ == token.ERROR {
				if sep != "" {
					fmt.Println()
				}
				log.Fatal(val)
			}
			fmt.Printf("%s%q[%s]", sep, val, typ)
			if typ == token.EOL || typ == token.COMMENT {
				sep = "\n"
			} else {
				sep = " "
			}
		}
		if sep != "" {
			fmt.Println()
		}
	}
}
