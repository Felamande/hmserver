package cookie

import (
	"net/http"
	"reflect"

	"github.com/go-martini/martini"
	"github.com/qiniu/log"
)

var constSalt string

func init() {
    
}




func Bind(cookieStruct interface{}) martini.Handler {
	return func(c martini.Context, req *http.Request, logger *log.Logger) {
		ensureNotPointer(cookieStruct)
		cookieStruct := reflect.New(reflect.TypeOf(cookieStruct))
		//获取cookie
		cookies := req.Cookies()
		var cookieMap = make(map[string]interface{}, len(cookies)+1)
		for _, cookie := range cookies {
			cookieMap[cookie.Name] = cookie.Value
		}

		if cookieStruct.Kind() == reflect.Ptr {
			cookieStruct = cookieStruct.Elem()
		}

		structTyp := cookieStruct.Type()

		for i := 0; i < structTyp.NumField(); i++ {
			typeField := structTyp.Field(i)
			structField := cookieStruct.Field(i)
			if typeField.Type.Kind() != reflect.String {
				panic("member of struct must be a string")
			}

			FieldTag := typeField.Tag.Get("cookie")
			if len(FieldTag) == 0 {
				FieldTag = typeField.Name
			}
			if cookieVal, exist := cookieMap[FieldTag]; exist {
				// if !structField.CanSet() {
				//     logger.Error("cant set!")
				// 	continue
				// }
				structField.Set(reflect.ValueOf(cookieVal))
			}

		}

		cookieStructParsed := cookieStruct.Interface()
		c.Map(cookieStructParsed)
		logger.Println(req.Method, req.URL.Path, "with cookie:", cookieMap)

	}
}

// func WriteCookie() martini.Handler{

// }

func ensureNotPointer(obj interface{}) {
	if reflect.TypeOf(obj).Kind() == reflect.Ptr {
		panic("Pointers are not accepted as binding models")
	}
}


