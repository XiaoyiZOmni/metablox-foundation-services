package controllers

import (
	"github.com/MetaBloxIO/metablox-foundation-services/comm/requtil"
	"github.com/MetaBloxIO/metablox-foundation-services/models"
	"github.com/MetaBloxIO/metablox-foundation-services/service"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

func GetNearbyMinersListHandler(c *gin.Context) {

	req, err := requtil.ShouldBindQuery[models.MinersReq](c)
	if err != nil {
		return
	}

	if req.Longitude.IsZero() || req.Longitude.IsZero() {
		ResponseErrorWithMsg(c, CodeError, errors.New("longitude\\&latitude are required"))
	}

	list, err := service.GetNearbyMinersList(&models.MinersDTO{
		Distance:  req.Distance,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	})

	if err != nil {
		ResponseErrorWithMsg(c, CodeError, err.Error())
		return
	}

	ResponseSuccess(c, list)
}
