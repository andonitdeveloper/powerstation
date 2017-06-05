package ids

import (
	"time"

	"github.com/andonitdeveloper/powerstation/ex"
	"sync"
	"github.com/andonitdeveloper/powerstation/code"
	"github.com/andonitdeveloper/log"
)
var logger = log.New()

var mapLock *sync.Mutex = &sync.Mutex{}

var channel chan int64 = make(chan int64, 100)

type syncHandler interface {
	downSync(string) (int64, code.Code)
	upSync(int64) (code.Code)
}


func GetId() (id int64, err error){
	timer := time.NewTimer(1 * time.Second)

	select{
	case id = <- channel:
	case <- timer.C:
		err = ex.TimeoutError{}
	}

	return
}

func Start(down func() int64, up func(int64) error){
	i := down()
}


