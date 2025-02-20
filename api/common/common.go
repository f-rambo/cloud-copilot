package common

import "github.com/pkg/errors"

func Response(failMsg ...string) *Msg {
	if len(failMsg) <= 0 {
		return &Msg{
			Reason:  ErrorReason_SUCCEED,
			Message: ErrorReason_name[int32(ErrorReason_SUCCEED.Number())],
		}
	}
	return &Msg{
		Reason:  ErrorReason_FAILED,
		Message: failMsg[0],
	}
}

func ResponseError(errMsg ErrorReason) error {
	if errMsg == ErrorReason_SUCCEED {
		return nil
	}
	return errors.New(errMsg.String())
}
