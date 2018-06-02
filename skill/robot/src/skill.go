package examples

import (
	"math"
	"mind/core/framework/drivers/distance"
	"mind/core/framework/drivers/hexabody"
	"mind/core/framework/drivers/media"
	"mind/core/framework/log"
	"mind/core/framework/skill"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/lazywei/go-opencv/opencv"
)

const ALL_VIEWS_BUFFER_SIZE = 1000
const VIEW_EXPIRATION_IN_SECONDS = 300
const SIZE_OF_INTERVAL_IN_DEGREES = 60.0
const INTERVALS = 360 / SIZE_OF_INTERVAL_IN_DEGREES
const TIME_TO_COMPLETE_MOVEMENT = 200
const TIME_TO_SLEEP_AFTER_MOVEMENT_IN_MS = time.Millisecond * 200
const GROUND_TO_FACE_PITCH_ANGLE = 20.0

type FollowSkill struct {
	skill.Base
	state           FollowState
	stop            chan bool
	allViews        chan View
	viewsWithFaces  chan View
	adjustView      chan View
	targetDirection float64
}

func NewSkill() skill.Interface {
	return &FollowSkill{
		state:           FollowState{"idle"},
		stop:            make(chan bool),
		allViews:        make(chan View, ALL_VIEWS_BUFFER_SIZE),
		viewsWithFaces:  make(chan View),
		adjustView:      make(chan View),
		targetDirection: 0,
	}
}

type FollowState struct {
	currState string
}

/*====================================================
EVENTS
=====================================================*/

func (FS *FollowSkill) OnStart() {
	log.Info.Println("Started")
	hexabody.Start()
	if !media.Available() {
		log.Error.Println("Media driver not available")
		return
	}
	if err := media.Start(); err != nil {
		log.Error.Println("Media driver could not start")
	}
	connectToServer()
}

func (FS *FollowSkill) OnClose() {
	hexabody.Close()
}

func (FS *FollowSkill) OnDisconnect() {
	os.Exit(0) // Closes the process when remote disconnects
}

func (FS *FollowSkill) OnRecvString(data string) {
	log.Info.Println(data)
	switch data {
	case "test":
		sendDataToServer()
		break
	case "start":
		break
	case "stop":
		FS.stop <- true
		break
	case "pic":
		PitchTest()
		break
	case "spinAround":
		go FS.FollowAsync()
		break
	}
}

/*====================================================
End Events
====================================================*/
func connectToServer() net.Conn {
	conn, err := net.Dial("tcp", "10.0.0.85:8080")
	if err != nil {
		log.Error.Println(err)
		// handle error
	}
	// fmt.Fprintf(conn, "GET / HTTP/1.0\r\n\r\n")
	//status, err := bufio.NewReader(conn).ReadString('\n')
	//log.Info.Println(status)
	return conn
}

func sendDataToServer() {
	conn := connectToServer()
	conn.Write([]byte("test message" + "\n"))
	conn.Close()
}

/*
LookAround
State: working
Description: turn head in 360 and take pictures which are pushed into the all views channel
*/
func (FS *FollowSkill) LookAround() {
	FS.state.currState = "searching"
	currentInterval := 0

	direction := LookAt2(0.0, GROUND_TO_FACE_PITCH_ANGLE) // start from same spot everytime
	if direction == -1 {
		log.Error.Println("look at failed")
		return
	}
	for FS.state.currState == "searching" {
		select {
		case <-FS.stop:
			log.Info.Println("stop received")
			break
		default:
			FS.allViews <- NewView("LookAround-"+strconv.Itoa(int(direction)), TakePic(), direction, GROUND_TO_FACE_PITCH_ANGLE, time.Now())

			if math.Mod(float64(currentInterval), float64(INTERVALS)) == (INTERVALS - 1) {
				return
			}
			currentInterval = currentInterval + 1
			direction = LookAt2(SIZE_OF_INTERVAL_IN_DEGREES*float64(currentInterval), GROUND_TO_FACE_PITCH_ANGLE)
			if direction == -1 {
				log.Error.Println("look at failed")
				break
			}

		}
	}
	logger("LookAround Complete")
	return
}

func (FS *FollowSkill) ContainsFaceAsync(view View) {
	log.Info.Println("Time since captured: ", time.Now().Sub(view.timestamp))
	var cvimg *opencv.IplImage
	var faces []*opencv.Rect
	if view.image == nil {
		log.Error.Println("NO IMAGE")
	}
	cvimg = opencv.FromImage(view.image)
	if cvimg == nil {
		log.Error.Println("NO CVIMG")
	}

	faces = opencv.LoadHaarClassifierCascade("assets/haarcascade_frontalface_alt.xml").DetectObjects(cvimg) //maybe cant be in go routine?
	if len(faces) > 0 {
		log.Info.Println("******Face found at ", "view: ", view.name+"-", view.direction)
		SendImage(view.image)
		FS.viewsWithFaces <- view
	}
	log.Info.Println("no faces found in ", view.id)
}

func (FS *FollowSkill) FindFaces() {
	for {
		select {
		case <-FS.stop:
			log.Info.Println("stop called during find faces")
			return
		case currentView := <-FS.allViews:
			if int(time.Now().Sub(currentView.timestamp).Seconds()) > VIEW_EXPIRATION_IN_SECONDS {
				log.Info.Println("too long since taken, image has expired")
				break
			}
			cV := currentView
			go func() {
				FS.ContainsFaceAsync(cV)
			}()
		}
	}
}

func (FS *FollowSkill) CheckPeripherals(view View) {
	log.Info.Println("Adjusting direction")
	if ContainsFace(look(view, 1).image) == true {
		log.Info.Println("success on look left")
		FS.viewsWithFaces <- look(view, 1)
	} else if ContainsFace(look(view, -1).image) == true {
		log.Info.Println("success on look right")
		FS.viewsWithFaces <- look(view, -1)
	} else {
		log.Info.Println("could not relocate face")
		go FS.LookAround()
	}
}

func (FS *FollowSkill) ConfirmFaceFound() {
	for {
		select {
		case <-FS.stop:
			hexabody.StopWalkingContinuously()
			logger("stop called")
			return
		case viewWithFace := <-FS.viewsWithFaces:
			log.Info.Println("looking at view: ", viewWithFace.id)
			look(viewWithFace, 0)
			log.Info.Println("calculated direction: ", viewWithFace.direction, " API direction: ", hexabody.Direction())
			image := TakePicAndSend()
			lastView := View{viewWithFace.id, "ConfirmFaceFound-" + strconv.Itoa(int(viewWithFace.direction)), image, viewWithFace.direction, viewWithFace.angle, time.Now()}
			if ContainsFace(lastView.image) {
				FS.state.currState = "following"
				FS.targetDirection = lastView.direction
				log.Info.Println("Success!")

			} else {
				FS.CheckPeripherals(lastView)
			}
			//os.Exit(0)
			//hexabody.Walk(viewWithFace.direction, 5000)
			//close channel if found? stop go routines to see if look at works
		}
	}
}

func (FS *FollowSkill) MoveToTarget() {
	for {
		select {
		case <-FS.stop:
			log.Info.Println("stop received")
			return

		default:
			switch {
			case FS.state.currState == "following":
				distance.Start()
				dist, err := distance.Value()
				if err != nil {
					log.Info.Println("error reading distance")
				}
				distance.Close()
				log.Info.Println(" distance: ", dist)
				log.Info.Println("following ", FS.targetDirection)
				hexabody.MoveHead(FS.targetDirection, 100)
				hexabody.Walk(FS.targetDirection, 50)
				break
			default:
				break
			}
			break
		}
	}
}

//Follow Skill is the entry point for this file
func (FS *FollowSkill) FollowAsync() {
	if FS == nil {
		log.Info.Println("no follow skill")
	}

	StartingDirection := hexabody.Direction()
	log.Info.Println("Current Direction: ", StartingDirection)
	go FS.LookAround()
	go FS.FindFaces()
	go FS.ConfirmFaceFound()
	go FS.MoveToTarget()
	//search
	//moveTowards
	//maintainDistance
}

//Follow()
///AnalyzeSurroundings()
////CaptureSurroundings()
////FindFaces()
///TrackTarget()
////CheckLastKnownView()
