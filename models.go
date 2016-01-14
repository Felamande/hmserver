package main

import (
	"reflect"
	"regexp"
    "mime/multipart"
)

//fkey=md5(md5(pwd+priv)+const)

type User struct {
	Name   string `cookie:"name" sql:"name"`
	SecKey string `cookie:"fk" sql:"sec_key"`
	Priv   string `cookie:"p" sql:"priv"`
}

type UserKey struct {
	SecKey string `cookie:"fk" sql:"sec_key"`
	Priv   string `cookie:"p" sql:"priv"`
}

func (k UserKey) Validate() bool {
	return len(k.SecKey) != 0 && len(k.Priv) != 0
}

type UserLoginForm struct {
	Pwd  string `form:"pwd" binding:"required"`
	Name string `form:"name" binding:"required"`
}

func (f UserLoginForm) Validate() bool {
	return len(f.Pwd) != 0 && len(f.Name) != 0
}

type UserCookie struct {
	Name   string `cookie:"name" sql:"name"`
	SecKey string `cookie:"fk" sql:"sec_key"`
	Priv   string `cookie:"p" sql:"priv"`
}

func (cookie UserCookie) Validate() bool {
	v := reflect.ValueOf(cookie)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Interface().(string) == "" {
			return false
		}
	}
	return true
}

type Sched struct {
	Year  int  `form:"year" json:"year" binding:"required" sql:"year"`
	Month int8 `form:"month" json:"month" binding:"required" sql:"month"`
	Day   int8 `form:"day" json:"day" binding:"required" sql:"day"`

	Hour   int8 `form:"hour" json:"hour" binding:"required" sql:"hour"`
	Minute int8 `form:"minute" json:"minute" binding:"required" sql:"minute"`

	Place string `form:"place" json:"place" sql:"place"`
	Event string `form:"event" json:"event" sql:"event"`

	Repeat int `form:"repeat" json:"repeat" binding:"required" gorm:"column:repeattype"`
}

func (s Sched) Validate() bool {
	return s.Month >= 1 && s.Month <= 12 && s.Day >= 1 && s.Day <= 21 && s.Hour >= 0 && s.Hour <= 24 && s.Minute >= 0 && s.Minute <= 59 && s.Repeat >= 0 && s.Repeat <= 2
}

type UserRegisterForm struct {
	Pwd  string `form:"pwd" binding:"required"`
	Name string `form:"name" binding:"required"`
}

func (f UserRegisterForm) Validate() bool {
	matched, _ := regexp.MatchString(`^[a-z|A-Z|0-9]+$`, f.Name)
	return len(f.Name) <= 10 && matched

}

type Bkimg struct{
    Content *multipart.FileHeader   `form:"bkimg" binding:"required"` 
}

type J map[string]interface{}
