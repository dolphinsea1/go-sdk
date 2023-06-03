package compare

import (
	"encoding/json"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/tidwall/gjson"
	"reflect"
	"strconv"
	"strings"
)

type DiffReport struct {
	Field   string      `json:"field"`
	OldData interface{} `json:"old_data,omitempty"`
	NewData interface{} `json:"new_data,omitempty"`
	Flag    string      `json:"-"`
}

type DiffRes struct {
	Action  string      `json:"action"` // ADD  MOD DEL
	Field   string      `json:"field,omitempty"`
	OldData interface{} `json:"old_data,omitempty"`
	NewData interface{} `json:"new_data,omitempty"`
	Diff    []DiffRes   `json:"diff"`
}

// DiffReq 请求结构
type DiffReq struct {
	JsonOld     string   `json:"json_old"`
	JsonNew     string   `json:"json_new"`
	DeLevel     int      `json:"de_level"`
	KeyField    []string `json:"key_field"` //比对array时使用
	IgnoreField []string `json:"ignore_field"`
}

const (
	Modify = "MOD"
	Add    = "ADD"
	Delete = "DEL"
)

func CompareWithLevel(json1, json2 string, level int, keyField, ignoreField []string) ([]DiffRes, error) {
	// 解析JSON字符串
	oldJson := gjson.Parse(json1)
	newJson := gjson.Parse(json2)
	res := make([]DiffRes, 0)
	diffList(oldJson, newJson, level, keyField, ignoreField, &res)

	fmt.Println(res)
	return res, nil
}

func diffList(oldJson, newJson gjson.Result, level int, keyField, ignoreField []string, reports *[]DiffRes) {
	if level <= 1 {
		if !cmp.Equal(oldJson, newJson) {
			*reports = append(*reports, DiffRes{
				Action:  Modify,
				OldData: oldJson.Str,
				NewData: newJson.Str,
			})
		}
	}
	// 获取所有键名

	oldField := make(map[string]interface{}, 0)
	recursiveJSONGet(oldJson.Str, &oldField, 0, level)
	newField := make(map[string]interface{}, 0)
	recursiveJSONGet(oldJson.Str, &oldField, 0, level)

	if level == 2 {
		for k, v := range oldField {
			if containsList(ignoreField, k) {
				continue
			}
			if v2, ok := newField[k]; ok {
				//modify
				diff := DiffRes{
					Action:  Modify,
					Field:   k,
					OldData: oldJson.Str,
					NewData: newJson.Str,
				}
				if reflect.DeepEqual(v, v2) {
					*reports = append(*reports, DiffRes{
						Action:  Modify,
						Field:   k,
						OldData: oldJson.Str,
						NewData: newJson.Str,
					})
				} else {
					//递归
					di := make([]DiffRes, 0)
					old1, _ := json.Marshal(v)
					new1, _ := json.Marshal(v2)
					diffList(gjson.Parse(string(old1)), gjson.Parse(string(new1)), level-1, keyField, ignoreField, &di)
					diff.Diff = di
				}
				*reports = append(*reports, diff)
			} else {
				//delete
				*reports = append(*reports, DiffRes{
					Action:  Delete,
					Field:   k,
					OldData: oldJson.Str,
				})
			}
		}

	}

	// 比对根据层级比对
	for k, v := range newField {
		//存在则忽略不对比
		if containsList(ignoreField, k) {
			continue
		}
		if _, ok := oldField[k]; !ok {
			*reports = append(*reports, DiffRes{
				Action:  Add,
				Field:   k,
				NewData: v,
			})
		}
	}
}

// 递归获取 JSON 对象的键值对，并存储到 map 中
// 递归获取 JSON 对象的键值对，并存储到 map 中
func recursiveJSONGet(jsonStr string, result *map[string]interface{}, depth int, maxDepth int) {
	jsonValue := gjson.Parse(jsonStr)
	jsonValue.ForEach(func(key, value gjson.Result) bool {
		if depth < maxDepth {
			if value.Type == gjson.JSON {
				subJSONStr := value.Raw
				subResult := make(map[string]interface{})
				recursiveJSONGet(subJSONStr, &subResult, depth+1, maxDepth)
				(*result)[key.String()] = subResult
			} else if value.IsArray() {
				subResult := make([]interface{}, 0)
				for _, arrValue := range value.Array() {
					if arrValue.Type == gjson.JSON {
						subJSONStr := arrValue.Raw
						subSubResult := make(map[string]interface{})
						recursiveJSONGet(subJSONStr, &subSubResult, depth+1, maxDepth)
						subResult = append(subResult, subSubResult)
					} else {
						subResult = append(subResult, arrValue.Value())
					}
				}
				(*result)[key.String()] = subResult
			} else {
				(*result)[key.String()] = value.Value()
			}
		} else {
			if value.Type == gjson.JSON || value.IsArray() {
				(*result)[key.String()] = "..."
			} else {
				(*result)[key.String()] = value.Value()
			}
		}
		return true
	})
}

// 递归函数，获取JSON对象内部所有层级的键名，包括数组下标
func getAllKeys(value gjson.Result, keys *[]string, maxLevel int, parentKeys ...string) {
	if value.IsObject() {
		*keys = append(*keys, concatenateKeys(parentKeys...))
		if 0 <= maxLevel {
			value.ForEach(func(key, value gjson.Result) bool {
				getAllKeys(value, keys, maxLevel-1, append(parentKeys, key.String())...)
				if maxLevel == 0 {
					return false
				}
				return true
			})
		}
	} else if value.IsArray() {
		*keys = append(*keys, concatenateKeys(parentKeys...))
		if 0 <= maxLevel {
			value.ForEach(func(idx, value gjson.Result) bool {
				getAllKeys(value, keys, maxLevel-1, append(parentKeys, strconv.Itoa(int(idx.Int())))...)
				if maxLevel == 0 {
					return false
				}
				return true
			})

		}
	} else {
		if 0 <= maxLevel {
			*keys = append(*keys, concatenateKeys(parentKeys...))
		}
	}
}

// 辅助函数，将所有键名拼接成一个字符串
func concatenateKeys(keys ...string) string {
	result := ""
	for _, key := range keys {
		if result != "" {
			result = fmt.Sprintf("%s.%s", result, key)
		} else {
			result = key
		}
	}
	return result
}

// 判断一个字符串是否在一个字符串数组中
func containsList(arr []string, str string) bool {
	split := strings.Split(str, ".")
	for _, s := range arr {
		for _, s2 := range split {
			if s == s2 {
				return true
			}
		}

	}
	return false
}
