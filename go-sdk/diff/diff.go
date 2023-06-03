package diff

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/tidwall/gjson"
	"reflect"
	"strings"
)

type DiffReport struct {
	Field   string      `json:"field"`
	OldData interface{} `json:"old_data,omitempty"`
	NewData interface{} `json:"new_data,omitempty"`
	Flag    string      `json:"-"`
}

type DiffRes struct {
	Action string       `json:"action"` // ADD  MOD DEL
	Diff   []DiffReport `json:"diff"`
}

// DiffReq 请求结构
type DiffReq struct {
	JsonOld     string   `json:"json_old"`
	JsonNew     string   `json:"json_new"`
	DeLevel     int      `json:"de_level"`
	KeyField    []string `json:"key_field"`
	IgnoreField []string `json:"ignore_field"`
}

const (
	Modify = "MOD"
	Add    = "ADD"
	Delete = "DEL"
)

// CompareJSON 比较两个 JSON 字符串，返回差异数据列表
func CompareJSON(req DiffReq) ([]DiffRes, error) {
	diffReports := make([]DiffReport, 0)
	var res []DiffRes

	oldObject := gjson.Parse(req.JsonOld)
	newObject := gjson.Parse(req.JsonNew)

	compare("", oldObject.Raw, newObject.Raw, req.DeLevel, req.KeyField, req.IgnoreField, &diffReports)
	m := make(map[string][]DiffReport, 0)
	for _, report := range diffReports {
		m[report.Flag] = append(m[report.Flag], report)
	}

	for k, v := range m {
		res = append(res, DiffRes{
			Action: k,
			Diff:   v,
		})
	}
	return res, nil
}

// compare 递归查询多节点的层级
func compare(fieldName string, oldVal, newVal interface{}, deLevel int, keyFields, ignoreFields []string, diffReports *[]DiffReport) {
	if deLevel == 1 {
		if !cmp.Equal(oldVal, newVal) {
			addDiffReport(fieldName, oldVal, newVal, diffReports)
		}
		return
	}

	oldKind := reflect.TypeOf(oldVal).Kind()
	newKind := reflect.TypeOf(newVal).Kind()

	if oldKind != newKind {
		addDiffReport(fieldName, oldVal, newVal, diffReports)
		return
	}

	//依据JSON类型进一步进行差异对比
	switch oldKind {
	case reflect.Map:
		oldMap := oldVal.(map[string]interface{})
		newMap := newVal.(map[string]interface{})

		for key, oldItem := range oldMap {
			newItem, ok := newMap[key]
			if !ok {
				addDiffReport(fmt.Sprintf("%s.%s", fieldName, key), oldItem, nil, diffReports)
				continue
			}
			if isKeyField(keyFields, fieldName, key) && !reflect.DeepEqual(oldItem, newItem) {
				compare(fmt.Sprintf("%s.%s", fieldName, key), oldItem, newItem, deLevel-1, keyFields, ignoreFields, diffReports)
			} else if !isIgnoreField(ignoreFields, fieldName) {
				compare(fmt.Sprintf("%s.%s", fieldName, key), oldItem, newItem, deLevel, keyFields, ignoreFields, diffReports)
			}
		}

		for key, newItem := range newMap {
			_, ok := oldMap[key]
			if !ok {
				addDiffReport(fmt.Sprintf("%s.%s", fieldName, key), nil, newItem, diffReports)
			}
		}
	case reflect.Slice, reflect.Array:
		oldSlice := oldVal.([]interface{})
		newSlice := newVal.([]interface{})

		if deLevel == 2 {
			if len(oldSlice) != len(newSlice) {
				addDiffReport(fieldName, oldVal, newVal, diffReports)
				return
			}

			for i := range oldSlice {
				if !reflect.DeepEqual(oldSlice[i], newSlice[i]) && !isIgnoreField(ignoreFields, fieldName) {
					addDiffReport(fmt.Sprintf("%s[%d]", fieldName, i), oldSlice[i], newSlice[i], diffReports)
				}
			}
		} else {
			for i, oldItem := range oldSlice {
				fmt.Println(i, oldItem)
				if len(newSlice) <= i {
					addDiffReport(fmt.Sprintf("%s[%d]", fieldName, i), oldItem, nil, diffReports)
					continue
				}
				compare(fmt.Sprintf("%s[%d]", fieldName, i), oldItem, newSlice[i], deLevel-1, keyFields, ignoreFields, diffReports)
			}

			for i := len(oldSlice); i < len(newSlice); i++ {
				addDiffReport(fmt.Sprintf("%s[%d]", fieldName, i), nil, newSlice[i], diffReports)
			}
		}
	default:
		if !reflect.DeepEqual(oldVal, newVal) {
			addDiffReport(fieldName, oldVal, newVal, diffReports)
		}
	}
}

// addDiffReport 添加差异数据
func addDiffReport(fieldName string, oldVal, newVal interface{}, diffReports *[]DiffReport) {
	action := Modify
	if oldVal == nil {
		action = Add
	}
	if newVal == nil {
		action = Delete
	}
	*diffReports = append(*diffReports, DiffReport{
		Field:   fieldName,
		OldData: oldVal,
		NewData: newVal,
		Flag:    action,
	})
}

// isKeyField 判断是否为需要对比key
func isKeyField(keyFields []string, fieldName string, key string) bool {
	if len(keyFields) == 0 {
		return true
	}

	for _, item := range keyFields {
		if strings.HasSuffix(item, key) || strings.HasSuffix(item, fmt.Sprintf(".%s", key)) ||
			strings.HasSuffix(item, fmt.Sprintf(".%s.", key)) || strings.HasPrefix(item, fieldName) {
			return true
		}
	}

	return false
}

// isIgnoreField 判断是否为忽略字段
func isIgnoreField(ignoreFields []string, fieldName string) bool {
	if len(ignoreFields) == 0 {
		return false
	}
	for _, item := range ignoreFields {
		fmt.Println(fieldName, item)
		if strings.Contains(fieldName, item) {
			return true
		}
	}

	return false
}
