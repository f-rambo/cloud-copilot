package common

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
