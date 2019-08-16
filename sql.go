package gomysql

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"strconv"
	"strings"
)

//var conn map[string]*sql.DB
var client map[string]*db

type db struct {
	conn *sql.DB
	conf string
}

var CONNECTDBERROR = errors.New("can't connect db")
var NotConnetKey = errors.New("can't found connect key")
var NotInitERROR = errors.New("not save conf")
var TAGERROR = errors.New("not found tag")

type Sqlconfig struct {
	UserName string
	Password string
	Host     string
	Port     int
	DbName   string
}

// 传入自定义连接字符串
func SaveStringConfAndConn(tag string, conf string) error {
	//判断是否是空map
	if client == nil {
		client = make(map[string]*db, 0)
	}
	if tag == "" {
		return TAGERROR
	}
	// 保存到配置
	client[tag].conf = conf


	db, err := sql.Open("mysql", client[tag].conf)
	if err != nil {
		return err
	}

	if err = db.Ping(); err != nil {
		return err
	}
	// 保存连接
	client[tag].conn = db
	return nil
}

// 如果tag 是空的, 那么默认dbname
func SaveConfAndConn(tag string, c *Sqlconfig) error {
	//判断是否是空map
	if client == nil {
		client = make(map[string]*db, 0)
	}
	connstring := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4",
		c.UserName, c.Password, c.Host, c.Port, c.DbName,
	)
	if tag == "" {
		client[c.DbName].conf = connstring
	} else {
		client[tag].conf = connstring
	}

	db, err := sql.Open("mysql", client[tag].conf)
	if err != nil {
		return err
	}

	if err = db.Ping(); err != nil {
		return err
	}
	client[tag].conn = db
	return nil
}

func ConnDB(tag string) error {



}


func GetConnections(tag string) int {
	return client[tag].conn.Stats().OpenConnections
}

func Update(tag,cmd string, args ...interface{}) (sql.Result, error) {
	if _, ok := client[tag]; !ok {
		return nil, NotConnetKey
	}
	if client[tag].conn == nil {
		// 重连
		if err := ConnDB(tag); err != nil {
			panic(err)
		}
	}
	if err := client[tag].conn.Ping(); err != nil {
		return nil, err
	}
	return client[tag].conn.Exec(cmd, args...)

}

func Insert(tag,cmd string, args ...interface{}) (sql.Result, error) {
	if _, ok := client[tag]; !ok {
		return nil, NotConnetKey
	}
	if client[tag].conn == nil {
		if err := ConnDB(tag); err != nil {
			panic(err)
		}
	}
	return client[tag].conn.Exec(cmd, args...)
}


func InsertMany(tag,cmd string, args []interface{}) (sql.Result, error) {

	if _, ok := client[tag]; !ok {
		return nil, NotConnetKey
	}
	if client[tag].conn == nil {
		if err := ConnDB(tag); err != nil {
			panic(err)
		}
	}
	if args == nil {
		return Insert(tag, cmd)
	}

		//找到括号的内容
	// 先转为为小写
	lowercmd := strings.ToLower(cmd)
	// 找到关键字 values
	tmp_index := strings.Index(lowercmd, " values")
	if tmp_index < 0 {
		return nil, errors.New("insert sql error")
	}
	// 找到关键字 后面的第一个 (
	start_index := strings.Index(cmd[tmp_index:], "(")
	if start_index < 0 {
		return nil, errors.New("sql error: eg: insert into table(name) values(?)")
	}
	end_index := strings.LastIndex(cmd, ")")
	if start_index < 0 {
		return nil, errors.New("sql error: eg: insert into table(name) values(?)")
	}
	value := cmd[tmp_index+start_index: end_index+1]
	//查看一行数据有多少列
	column := 0
	for _, v := range strings.Split(value, ",") {
		opt := strings.Trim(v, " ")
		if opt == "?" {
			column++
		}
	}

	// 总共多少参数
	count := len(args)
	if count % column != 0 {
		return nil, errors.New("args error")
	}
	addcmd := "," + value
	for i := 1; i < count % column; i++ {
		cmd = cmd + addcmd
	}

	return Insert(tag, cmd, args...)
}

func GetRows(tag,cmd string, args ...interface{}) (*sql.Rows, error) {
	if _, ok := client[tag]; !ok {
		return nil, NotConnetKey
	}
	if client[tag].conn == nil {
		if err := ConnDB(tag); err != nil {
			panic(err)
		}
	}
	return client[tag].conn.Query(cmd, args...)

}

func Close(tag string) {
	//存在并且不为空才关闭
	if _, ok := client[tag]; ok && client[tag] != nil {
		client[tag].conn.Close()
	}

}

func GetOne(tag,cmd string, args ...interface{}) *sql.Row {
	if _, ok := client[tag]; !ok {
		panic(NotConnetKey)
	}
	if client[tag].conn == nil {
		if err := ConnDB(tag); err != nil {
			panic(err)
		}
	}
	return client[tag].conn.QueryRow(cmd, args...)
}

// 还原sql
func cmdtostring(cmd string, args ...interface{}) string {

	var logstr string

	for _, v := range args {
		switch v.(type) {
		case int64:
			logstr = "'" + strconv.FormatInt(v.(int64), 10) + "'"
		case int:
			logstr = "'" + strconv.Itoa(v.(int)) + "'"
		case string:
			logstr = "'" + v.(string) + "'"
		default:
			logstr = "'" + v.(string) + "'"
			//return
		}
		cmd = strings.Replace(cmd, "?", "%s", 1)
		cmd = fmt.Sprintf(cmd, logstr)

	}
	return cmd
}