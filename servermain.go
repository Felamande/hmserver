package main

import (
    //标准库的包
	"net/http"
	"os"
	"path/filepath"
	"time"
    "io/ioutil"
	"bytes"
	"image"
    
    //引用的第三方开源包
	"github.com/go-martini/martini"
	"github.com/jinzhu/gorm"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"github.com/qiniu/log"
    _ "github.com/go-sql-driver/mysql"

    //本人写的包，即是src/cookie目录和src/util目录，Felamande是本人的github用户名。
	"github.com/Felamande/hmserver/cookie" 
	"github.com/Felamande/hmserver/util"

	
)

func main() {

	server := martini.Classic()
	config := InitConfig()

	//添加中间件
	server.Use(render.Renderer(render.Options{//render中间件可以把对象方便的序列化为xml或者json
		Directory:  "./html",
		Extensions: []string{".html"},
		Charset:    "UTF-8",
	}))
	server.Use(martini.Static("public"))//静态文件服务

	//添加路由
    //第一个参数是url路径，
    //之后的参数是处理该路径请求的处理函数，
    //可以添加多个，依次调用
    //方法名表示该路径的HTTP方法，表示只能用GET访问该路径。
	server.Get("/", cookie.Bind(UserCookie{}), handleHome)
    
    //Group是父路径下添加子路径，下面url分别表示/sched/all, /sched/all, /sched/delete。
    //在父路径里添加的处理函数在子路径中都会运行，
    //比如下面的cookie.Bind(UserCookie{})，该方法返回值是一个函数，表示这个路径以及所有的子路径都绑定了一个cookie。
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
		r.Group("/bkimg", func(rr martini.Router) {
			rr.Post("/upload", binding.MultipartForm(Bkimg{}), UploadBkimg)
			rr.Get("/get", GetBkimg)
		}, cookie.Bind(UserCookie{}))

	})

	//映射服务
	logger := log.New(os.Stdout, "[martini] ", log.Llevel|log.Lshortfile|log.Lmodule)
    //Map方法传入的对象可以被传入到处理函数的对应参数中。
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

//handleGetSched URL:/sched/all 获取日程表数据
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

//LoginHandler url: /user/login
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

//RegisterHandler url: /user/register
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

//CheckLoginHandler url: /user/checklogin
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
		r.JSON(http.StatusOK, J{"data": nil, "err": J{"code": 102, "msg": "invalid user"}})
		return
	}

	r.JSON(http.StatusOK, J{"data": cookie.Name})
}

//UploadBkimg url: /user/bkimg/upload
func UploadBkimg(img Bkimg, r render.Render, cookie UserCookie, config Config, logger *log.Logger) {
    //检查cookie的有效性
	if !cookie.Validate() {
		r.Redirect("/", http.StatusUnauthorized)
		logger.Info("Fail to auth whith cookie:", cookie)
		return
	}
    //打开上传文件
	file, err := img.Content.Open()
	if err != nil {
		r.Redirect("/", http.StatusInternalServerError)
		return
	}
    
    //将文件内容全被读出来
	b, err := ioutil.ReadAll(file)
	if err != nil {
		r.Redirect("/", http.StatusInternalServerError)
		return
	}
    
    //检查该图片文件的类型，如果不是图片文件的话那么上传失败，返回。
	_, format, err := image.Decode(bytes.NewReader(b))
	switch err {
	case image.ErrFormat:
		r.Redirect("/", http.StatusOK)
		return
	case nil:
		break
	default:
		r.Redirect("/", http.StatusInternalServerError)
		logger.Info(err.Error())
		return
	}
    
    //计算文件的md5，作为唯一表示以及文件名。
	fileMd5 := util.Md5(b)
	fileName := fileMd5 + "." + format
	fileFullName := filepath.Join(config.Server.StaticHome, "img/bk", fileName)
    
    //如果该文件存在那么直接跳到接入数据库
	if fi, _ := os.Stat(fileFullName); fi != nil {
		r.Redirect("/", http.StatusFound)
		logger.Info("file exists:", fileFullName)
		goto CommitToDB
	}
	err = ioutil.WriteFile(fileFullName, b, 0600)
	if err != nil {
		r.Redirect("/", http.StatusInternalServerError)
		return
	}
    
//将该图片文件的文件名存入users表的bkimg字段中。
CommitToDB:
	db, err := gorm.Open(config.DB.Type, config.DB.Uri)
	if err != nil {
		r.Redirect("/", http.StatusInternalServerError)
		return
	}

	if err = db.Table("users").Where(&cookie).Update("bkimg", fileName).Error; err != nil {
		r.Redirect("/", http.StatusInternalServerError)
		return
	}
	r.Redirect("/", http.StatusFound)

}

//GetBkimg url: /user/bkimg/get
func GetBkimg(cookie UserCookie, config Config, logger *log.Logger, r render.Render) {
	if !cookie.Validate() {
		r.JSON(http.StatusOK, J{"data": nil})
		logger.Info("Fail to auth whith cookie:", cookie)
		return
	}

	db, err := gorm.Open(config.DB.Type, config.DB.Uri)
	if err != nil {
		r.JSON(http.StatusInternalServerError, J{"data": nil, "err": J{"code": 201, "msg": "database open error."}})
		return
	}
	var BkimgName string

	row := db.Table("users").Where(&cookie).Select("bkimg").Row()

	if row == nil {
		r.JSON(http.StatusInternalServerError, J{"data": nil})
		logger.Error(err)
		return
	}

	row.Scan(&BkimgName)
	r.JSON(http.StatusOK, J{"data": BkimgName, "err": nil})

}
