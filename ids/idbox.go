package ids

import (
	"github.com/andonitdeveloper/powerstation/code"
	"time"
	"sync/atomic"
)

const(
	Ready = 1
	Down_Sync = 2
	Running = 3
	Up_Sync = 4
	Stop = 5
	InitialDelta = 10000
	MinDelta int64 = 1000
	IncreaseFactor = 1.2
	DecreaseFactor = 0.9
)

type IdResult struct{
	Id int64
	Code code.Code
}

type report struct{
	median int64
	startTime int64
}



type idBox struct{
	name string
	channel chan IdResult
	status int32
	i int64
	syncHdr syncHandler
	stop1 chan bool
	stop2 chan bool
	lim int64
	deltaChan chan int64
	reportChan chan report
}

func newIdBox(name string, syncHdr syncHandler) *idBox{
	return &idBox{
		name,
		make(chan IdResult, 1),
		Ready,
		0,
		syncHdr,
		make(chan bool, 1),
		make(chan bool, 1),
		0,
		make(chan int64, 20),
		make(chan report, 10),
	}
}

func (this *idBox) getStatus() int32{
	return atomic.LoadInt32(&this.status)
}

func (this *idBox) shutdown(){
	s := this.getStatus()
	if s == Ready || s == Stop{
		return
	}

	if atomic.CompareAndSwapInt32(&this.status, Running, Stop){
		this.stop1 <- true
		this.stop2 <- true
	}
}

func (this *idBox) downsync(cn chan int64){
	for{
		if num, c := this.syncHdr.downSync(this.name); c != code.OK{
			logger.Warnf("%s向本地同步时出错:%d", this.name, c)
			continue
		}else{
			cn <- num
			break
		}
	}
}

func (this *idBox) upsync(cn chan int64){
	num := this.i + InitialDelta
	for{
		status:= this.getStatus()
		if status == Stop{
			break
		}
		c:= this.syncHdr.upSync(num)
		if c!=code.OK{
			logger.Warnf("%s向consul同步时出错:%d", this.name, c)
		}else{
			cn <- num
			break
		}
	}
}

func (this *idBox) upsyncNum(num int64) bool{
	for{
		status:= this.getStatus()
		if status == Stop{
			break
		}
		c:= this.syncHdr.upSync(num)
		if c!=code.OK{
			logger.Warnf("%s向consul同步时出错:%d", this.name, c)
		}else{
			return true
		}


	}
	return false
}


func (this *idBox) guard(initNum int64){

	var totalTime int64 = 0
	var num int64 = 0
	var delta int64 = 0

	for{
		select{
		case <-this.stop2:
			return
		case rep:=<- this.reportChan:
			if totalTime != 0{
				t := totalTime / num
				t_cos := time.Now().Unix() - rep.startTime
				if t >= t_cos{
					delta = delta * (t/t_cos) * IncreaseFactor
				}else{
					delta = delta * DecreaseFactor
					if delta < MinDelta{
						delta = MinDelta
					}
				}
			}else{
				delta = MinDelta
			}

			start := time.Now().Unix()
			this.upsyncNum(initNum + delta)
			end :=time.Now().Unix()
			totalTime += start - end
			num ++
			initNum += delta
			this.deltaChan <- delta
		}
	}
}


func (this *idBox) run(){
	this.status = Down_Sync

	// down
	c1 := make(chan int64, 1)
	go this.downsync(c1)
	select{
	case <-this.stop1:
		return
	case this.i = <-c1:
	}

	// up

	c2 := make(chan int64, 1)
	go this.upsync(c2)
	select{
	case <- this.stop1:
		return
	case this.lim = <-c1:
	}

	go this.upsyncNum(this.lim)

	median := (this.lim - this.i)/2
	startTime := time.Now().Unix()

	for{
		select{
		case <- this.stop1:
			return
		case this.channel <- IdResult{ this.i, code.OK}:
			this.i = this.i + 1
			if this.i == median{
				this.reportChan <- report{median:median, startTime: startTime}
			}else if this.i == this.lim{
				select {
				case <- this.stop1:
					return
				case delta:=<-this.deltaChan:
					this.lim = this.lim + delta
					median = (this.lim - this.i)/2
					startTime = time.Now().Unix()
				}
			}
		}
	}

}
