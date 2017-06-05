package ex

const(
	TimeoutCode ErrorCode = 1
)

type ErrorCode uint8

type CodeError interface{
	Code() ErrorCode
}

type TimeoutError struct {

}

func (this *TimeoutError) Error() string{
	return "retrieve id timeout"
}

func (this *TimeoutError) Code() ErrorCode{
	return TimeoutCode
}
