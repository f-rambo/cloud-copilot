package biz

import "github.com/go-kratos/kratos/v2/log"

type EventUsecase struct {
	log *log.Helper
}

func NewEventUsecase(logger log.Logger) *EventUsecase {
	return &EventUsecase{
		log: log.NewHelper(logger),
	}
}
