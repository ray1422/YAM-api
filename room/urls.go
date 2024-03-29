package room

import (
	"github.com/gin-gonic/gin"
	"github.com/ray1422/YAM-api/room/signaling"
)

// RegisterRouter RegisterRouter
func RegisterRouter(roomGroup *gin.RouterGroup) {
	roomViews(roomGroup, "/")
	signaling.RoomWS(roomGroup, "/")
}
