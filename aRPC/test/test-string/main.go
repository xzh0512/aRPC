package main

import (
	"fmt"
	"strings"
)

func main() {
	str := "foo.bar.baz"

	// 使用点号（.）来划分字符串
	parts := strings.Split(str, ".")

	// 打印划分后的字符串切片
	fmt.Println(parts[0])
}
