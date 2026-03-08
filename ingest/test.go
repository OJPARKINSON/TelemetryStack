package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

func main() {
	file, err := os.Open("go/ibt_files/mclaren720sgt3_lemans full 2025-06-04 20-16-20.ibt")
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	fmt.Println("file success, ", file.Name())

	buf := make([]byte, 4)

	_, err = io.ReadFull(file, buf)
	if err != nil {
		fmt.Println(err)
	}

	// fmt.Println("buf size: ", binary.LittleEndian.Uint32(buf[122:143]))
	fmt.Println("buf success, ", binary.LittleEndian.Uint32(buf[0:4]))

}
