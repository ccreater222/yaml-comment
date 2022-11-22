package main

import (
	"fmt"
	yaml "github.com/ccreater222/yaml-comment"
)

type Test struct {
	Name     string `yaml:"name" head_comment:"head comment"`
	Nickname string `yaml:"nickname" line_comment:"line comment"`
	Age      int    `yaml:"age" foot_comment:"foot comment"`
}

func main() {
	t := Test{
		Name:     "a",
		Nickname: "b",
		Age:      0,
	}
	out, _ := yaml.Marshal(t)
	fmt.Println(string(out))
}
