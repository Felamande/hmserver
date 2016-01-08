
//当document已经完全在浏览器中呈现时候调用函数，否则无法获得页面中的html元素


var repeatType = { 0: "每日重复", 1: "每周重复", 2: "每月重复" }

$(document).ready(function () {
    
    //通过验证cookie检查用户是否已经登陆，如果已经登录就把登录框隐藏。
    $.post("/user/checklogin", null, function (json) {
        if (json.data) {
            $("#login-form,#login-btn").css("display", "none")
            $("#text-login").html("<br />" + json.data)
        }
    })
    
    //定义页面上几个item的标题，内容等属性，传入initItems函数中添加到页面上
    var funcItems = [
        {
            title: "我的日程",
            titleWrap: "h3",
            body: datagrid,//这是一个函数，用来定义这个item的内容是什么。
            media: [],
        },
        {
            title: "我的图片",
            titleWrap: "h2",
            body: imageLoop,
            media: [],
        },
        {
            title: "title 2",
            body: ["This is content of sec 2", "this is another content"],
            media: [],
        }
    ]

    initItems(funcItems)
    
    //ajax post登录
    $("#login-btn").click(function () {
        var params = $("#login-form").serialize()
        console.log(params)
        $.post("/user/login?" + params, null, function (json) {
            if (json.err) {
                alert(json.err.msg)
                return
            }
            $("#login-form,#login-btn").css("display", "none")
            $("#text-login").html("<br />" + json.data)
        })
    })
    //设置面板，可以改变页面风格
    var panel = $("#configure-panel")

    $("#configure-wrap").click(function(){
        panel.slideToggle()
    })
    
    
    $("#color-slider").slider({
        orientation:"horizontal",
        ragne:"min",
        max:255,
        min:0
        // value:127,
        // slide:changeColor,
        // change:changeColor
    })

})

function initItems(funcItems) {

    var funcSec = $("#function-section")

    for (var i in funcItems) {
        //用jquery创建一个item，用div标签。
        var itemWrapper = $("<div class='function-item'></div>")
        itemWrapper.attr({ id: "function-item-" + i })

        var item = funcItems[i]
        if (item.title) {
            //如果存在标题的话，就把标题加入右边的导航栏中。
            anchor(item.title, i)
            
            //创建一个标题，并且用titleWrap(一般是h1，h2等)包裹标题文字。
            var titleSec = $("<section class='function-title'></section>")
            if (item.titleWrap) {
                var title = $("<" + item.titleWrap + ">" + "</" + item.titleWrap + ">")
                title.html(item.title)
                title.appendTo(titleSec)
            } else {
                //如果没有定义titleWrap的话就把标题文字直接加入到标题的section中
                titleSec.html(item.title)
            }
            titleSec.appendTo(itemWrapper)
        }
        
        //如果定义了内容的话，item.body是一个函数，把这个jquery对象作为参数传进去。
        if (item.body) {
            var bodySec = $("<section class='function-body'></section>")
            if (typeof item.body == "function") {
                item.body(bodySec)
            }
            bodySec.appendTo(itemWrapper)
        }
        //最后把有标题和内容的一个item加入到页面上
        itemWrapper.appendTo(funcSec)
    }

}

function datagrid(content) {
    //从url /sched/all 获得数据，形成表格。
    $.get("/sched/all", null, function (json) {

        if (!json.data) {
            return
        }
        var data = json.data
        for (i in data) {
            data[i].date = data[i].year + "." + data[i].month + "." + data[i].day
            data[i].time = data[i].hour + ":" + data[i].minute
            data[i].repeat = repeatType[data[i].repeat]
        }
        content.datagrid({
            data: data,
            col: [{
                field: "date",
                title: "日期",
                sortable: false
            }, {
                    field: "time",
                    title: "时间",
                    sortable: false
                }, {
                    field: "event",
                    title: "事件",
                    sortable: true
                }, {
                    field: "repeat",
                    title: "重复类型",
                    sortable: true
                }],
            attr: { "class": "table table-bordered table-condensed" },
            sorter: "bootstrap"


        });
    })
}

function article(content) {
    
}

function imageLoop(content){
    
}

function anchor(text, index) {
    var anchorItem = $("<div class='anchor-item'><a></a></div>")
    var a = anchorItem.children("a")
    a.html(text)
    a.attr({ href: "#" + "function-item-" + index })
    anchorItem.appendTo("#anchor")
}
