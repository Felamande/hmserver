package main

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-martini/martini"
	"github.com/jinzhu/gorm"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"github.com/qiniu/log"

	"github.com/Felamande/hmserver/cookie"
	"github.com/Felamande/hmserver/util"

	"io/ioutil"

	"bytes"
	"image"

	_ "github.com/go-sql-driver/mysql"
)

func main() {

	server := martini.Classic()
	config := InitConfig()

	//添加中间件
	server.Use(render.Renderer(render.Options{
		Directory:  "./html",
		Extensions: []string{".html"},
		Charset:    "UTF-8",
	}))
	server.Use(martini.Static("public"))

	//添加路由
	server.Get("/", cookie.Bind(UserCookie{}), handleHome)

	server.Group("/sched", func(r martini.Router) {
		r.Post("/add", binding.Form(Sched{}), handleAddSched)
		r.Get("/all", handleGetSched)
		r.Post("/delete", handleDelSched)
	}, cookie.Bind(UserCookie{}))

	server.Group("/user", func(r martini.Router) {
		r.Post("/login", binding.Form(UserLoginForm{}), LoginHandler)
		r.Post("/logout")
		r.Post("/register", binding.Form(UserRegisterForm{}), RegisterHandler)
		r.Post("/checklogin", cookie.Bind(UserCookie{}), CheckLoginHandler)
		r.Group("/upload", func(rr martini.Router) {
			rr.Post("/bkimg", binding.MultipartForm(Bkimg{}), UploadBkimg)
		}, cookie.Bind(UserCookie{}))

	})

	//映射服务
	logger := log.New(os.Stdout, "[martini] ", log.Llevel|log.Lshortfile|log.Lmodule)
	server.Map(logger)
	server.Map(config)

	server.RunOnAddr(":" + config.Server.Port)

}

//handleHome URL:/
func handleHome(r render.Render, cookie UserCookie, logger *log.Logger) {
	r.HTML(http.StatusOK, "home", nil)
}

//handleAddSched URL: /sched/add
func handleAddSched(cookie UserCookie, schedForm Sched, r render.Render, logger *log.Logger, config Config) {
	if !cookie.Validate() {
		r.JSON(http.StatusOK, J{"data": nil})
		return
	}
	if !schedForm.Validate() {
		r.JSON(http.StatusOK, J{"data": nil, "err": J{"code": 101, "msg": "invalid form"}})
		return
	}
	db, err := gorm.Open(config.DB.Type, config.DB.Uri)
	if err != nil {
		r.JSON(http.StatusInternalServerError, J{"data": nil})
		logger.Error(err)
		return
	}
	defer db.Close()
	var count int
	db.Table("users").Where(&cookie).Count(&count)
	if count == 0 {
		r.JSON(http.StatusOK, J{"data": nil})
		return
	}
	err = db.Table("scheds").Create(&schedForm).Error
	if err != nil {
		r.JSON(http.StatusOK, J{"data": nil, "err": J{"code": 300, "msg": err.Error()}})
		return
	}

	r.JSON(http.StatusOK, J{"data": "insert OK"})

}

//handleGetSched URL:/sched/all
func handleGetSched(r render.Render, logger *log.Logger, config Config, cookie UserCookie) {

	if !cookie.Validate() {
		logger.Info("Fail to auth whith cookie:", cookie)
		r.JSON(http.StatusOK, J{"data": nil})
		return
	}

	//type表示数据库的类型，如mysql,sqlite3等
	//uri为需要打开的数据库连接，格式为user:password@/dbname?charset=utf8
	//两者都定义在config.ini中
	db, err := gorm.Open(config.DB.Type, config.DB.Uri)
	if err != nil {
		r.JSON(http.StatusInternalServerError, J{"data": nil})
		logger.Error(err)
		return
	}
	defer db.Close()
	var count int
	db.Table("users").Where(&cookie).Count(&count)
	if count == 0 {
		r.JSON(http.StatusOK, J{"data": nil})
		return
	}

	var sched []Sched
	db.Table("scheds").Select("*").Where("user=?", cookie.Name).Find(&sched)

	r.JSON(http.StatusOK, J{"data": sched})

	logger.Info("Schedule items total", len(sched), "in JSON,", "with cookie:", cookie)
}
func handleDelSched(r render.Render, logger *log.Logger) {

}

func LoginHandler(w http.ResponseWriter, config Config, form UserLoginForm, r render.Render, logger *log.Logger) {
	if !form.Validate() {
		r.JSON(http.StatusOK, J{"data": nil, "err": J{"code": 101, "msg": "invalid form"}})
		return
	}

	db, err := gorm.Open(config.DB.Type, config.DB.Uri)
	if err != nil {
		r.JSON(http.StatusInternalServerError, J{"data": nil, "err": J{"code": 201, "msg": "database open error."}})
		return
	}
	userKey := UserKey{}
	db.Table("users").Select("sec_key, priv").Where("name = ?", form.Name).First(&userKey)
	if !userKey.Validate() {
		r.JSON(http.StatusOK, J{"data": nil, "err": J{"code": 102, "msg": "unregistered user"}})
		return
	}

	p1 := util.Md5(form.Pwd, userKey.Priv)
	SecKey := util.Md5(p1, config.AuthConfig.ConstSalt)
	if SecKey != userKey.SecKey {
		r.JSON(http.StatusOK, J{"data": nil, "err": J{"code": 103, "msg": "invalid password"}})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "name",
		Value:   form.Name,
		Path:    "/",
		Expires: time.Now().Add(time.Hour * 10000),
	})
	http.SetCookie(w, &http.Cookie{
		Name:    "fk",
		Value:   userKey.SecKey,
		Path:    "/",
		Expires: time.Now().Add(time.Hour * 10000),
	})
	http.SetCookie(w, &http.Cookie{
		Name:    "p",
		Value:   userKey.Priv,
		Path:    "/",
		Expires: time.Now().Add(time.Hour * 10000),
	})

	r.JSON(http.StatusOK, J{"data": form.Name, "err": nil})

}

func RegisterHandler(w http.ResponseWriter, config Config, form UserRegisterForm, r render.Render, logger *log.Logger) {
	if !form.Validate() {
		r.JSON(http.StatusOK, J{"data": nil, "err": J{"code": 100, "msg": "invalid name"}})
		return
	}

	priv := util.GetRandomString(10)
	p1 := util.Md5(form.Pwd, priv)
	SecKey := util.Md5(p1, config.AuthConfig.ConstSalt)
	db, err := gorm.Open(config.DB.Type, config.DB.Uri)
	if err != nil {
		r.JSON(http.StatusInternalServerError, J{"data": nil, "err": J{"code": 201, "msg": "database open error."}})
		return
	}
	defer db.Close()

	NewUser := User{
		Name:   form.Name,
		SecKey: SecKey,
		Priv:   priv,
	}
	//把新用户插入users表中
	err = db.Table("users").Create(&NewUser).Error
	if err != nil {
		r.JSON(http.StatusInternalServerError, J{"data": nil, "err": J{"code": 202, "msg": "database insert error."}})
		logger.Error(err)
		return
	}

	r.JSON(http.StatusOK, J{"data": NewUser.Name, "err": nil})

}

func CheckLoginHandler(cookie UserCookie, r render.Render, config Config) {
	if !cookie.Validate() {
		r.JSON(http.StatusOK, J{"data": nil})
		return
	}
	db, err := gorm.Open(config.DB.Type, config.DB.Uri)
	if err != nil {
		r.JSON(http.StatusInternalServerError, J{"data": nil, "err": J{"code": 201, "msg": "database open error."}})
		return
	}
	defer db.Close()

	var count int
	db.Table("users").Where(&cookie).Count(&count)
	if count == 0 {
		r.JSON(http.StatusOK, J{"data": nil})
		return
	}

	r.JSON(http.StatusOK, J{"data": cookie.Name})
}

func UploadBkimg(img Bkimg, r render.Render, cookie UserCookie, config Config, logger *log.Logger) {
	if !cookie.Validate() {
		logger.Info("Fail to auth whith cookie:", cookie)
		r.JSON(http.StatusOK, J{"data": nil})
		return
	}
	file, err := img.Content.Open()
	if err != nil {
		r.JSON(http.StatusInternalServerError, J{"data": nil})
		return
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		r.JSON(http.StatusInternalServerError, J{"data": nil})
		return
	}
	_, format, err := image.Decode(bytes.NewReader(b))
	switch err {
	case image.ErrFormat:
		r.JSON(http.StatusOK, J{"data": nil, "err": J{"code": 400, "msg": "invalid format"}})
		return
	case nil:
		break
	default:
		r.JSON(http.StatusInternalServerError, J{"data": nil})
		logger.Info(err.Error())
		return
	}

	fileMd5 := util.Md5(b)
	fileName := filepath.Join(config.Server.StaticHome, "img/bk", fileMd5+"."+format)

	if fi, _ := os.Stat(fileName); fi != nil {
		r.JSON(http.StatusOK, J{"data": "upload ok", "err": nil})
		logger.Info("file exists:", fileName)
		goto CommitToDB
	}
	err = ioutil.WriteFile(fileName, b, 0600)
	if err != nil {
		r.JSON(http.StatusInternalServerError, J{"data": nil})
		return
	}

CommitToDB:
	db, err := gorm.Open(config.DB.Type, config.DB.Uri)
	if err != nil {
		r.JSON(http.StatusInternalServerError, J{"data": nil})
		return
	}

	if err = db.Table("users").Where(&cookie).Update("bkimg", fileMd5).Error; err != nil {
		r.JSON(http.StatusInternalServerError, J{"data": nil})
		return
	}
	r.JSON(http.StatusOK, J{"data": "upload ok", "err": nil})

}
