package go_sdk

import (
	"encoding/json"
	"fmt"
	"go-sdk/compare"
	"go-sdk/diff"
	"testing"
)

const json1 = `[{
		"name": "张so",
		"age": 18,
		"hobbies": ["篮球", "游泳", "旅游"]
	},
	{
		"name": "李四",
		"age": 20,
		"hobbies": ["足球", "游戏"]
	}
]`

const json2 = `[{
		"name": "张三",
		"age": 20,
		"hobbies": ["篮球", "游泳", "旅游", "阅读"]
	},
	{
		"name": "李四",
		"age": 20,
		"hobbies": ["足球", "游戏", "阅读"]
	},
	{
		"name": "王五",
		"age": 22,
		"hobbies": ["音乐", "电影"]
	}
]`

func Test1(t *testing.T) {

	// 解析JSON对象
	req := diff.DiffReq{
		JsonOld:     json1,
		JsonNew:     json2,
		DeLevel:     2,
		KeyField:    []string{"name"},
		IgnoreField: []string{"hobbies"},
	}
	res, _ := diff.CompareJSON(req)

	marshal, err := json.Marshal(res)
	if err != nil {
		fmt.Errorf("err:%v", err)
	}
	fmt.Println(string(marshal))
}

func Test2(t *testing.T) {
	res, _ := compare.CompareWithLevel(json1, json2, 1, []string{}, []string{})
	bytes, _ := json.Marshal(res)
	fmt.Println()
	fmt.Println(string(bytes))
}
