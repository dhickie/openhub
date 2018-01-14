package tv

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/dhickie/go-lgtv/control"
	"github.com/dhickie/hickhub/log"
	"github.com/dhickie/hickhub/messaging"
	"github.com/dhickie/hickhub/messaging/payloads"
	"github.com/dhickie/hickhub/models"
)

// ErrCommandUnsupported is returned when a requested command is not supported by the TV
var ErrCommandUnsupported = errors.New("Requested action is unsupported")

// tvController controlls all TVs under its remit when an appropriate message is received
type tvController struct {
	Tvs        map[string]*control.LgTv
	ClientKeys map[string]string
}

// subscriber is the callback called when the TV module receives a message
func (c *tvController) subscriber(msg messaging.Message) {
	// We know this is a command message, so unmarshal the payload as such
	cmd := new(payloads.CommandPayload)
	err := json.Unmarshal([]byte(msg.Payload), cmd)
	if err != nil {
		log.Error(fmt.Sprintf("An error occured unmarshalling the command payload: %v", err))
		return
	}

	// Perform the provided command on the TV with the given device ID
	success := false
	errStr := ""
	deviceState := models.DeviceState{}

	if tv, ok := c.Tvs[cmd.DeviceID]; ok {
		switch cmd.State {
		case models.StateVolume:
			err = handleVolumeCommand(tv, cmd.Command, cmd.Detail)
			if err == nil {
				var volState models.VolumeState
				volState, err = getVolumeState(tv)

				deviceState.Type = models.StateVolume
				deviceState.State = volState
			}
		case models.StateChannel:
			err = handleChannelCommand(tv, cmd.Command, cmd.Detail)
			if err == nil {
				var chanState models.ChannelState
				chanState, err = getChannelState(tv)

				deviceState.Type = models.StateChannel
				deviceState.State = chanState
			}
		case models.StatePower:
			err = handlePowerCommand(tv, cmd.Command, cmd.Detail)
			if err == nil {
				powerOn := false
				if cmd.Command == models.CommandOn {
					powerOn = true
				}
				deviceState.Type = models.StatePower
				deviceState.State = models.PowerState{PowerOn: powerOn}
			}
		}

		if err != nil {
			errStr = fmt.Sprintf("An error occured performing the requested TV operation: %v", err)
			log.Error(errStr)
		} else {
			success = true
		}
	} else {
		errStr = fmt.Sprintf("Received message for unknown device ID: %v", cmd.DeviceID)
		log.Error(errStr)
	}

	// Build the result payload and send it back
	resultMessage, err := messaging.NewCommandResultMessage(success, errStr, deviceState)
	if err != nil {
		// Log the error, we can't publish the result back :(
		log.Error(fmt.Sprintf("An error occured trying to create the result message: %v", err))
		return
	}

	messaging.Publish(msg.Reply, resultMessage)
}

func handleVolumeCommand(tv *control.LgTv, command string, detail string) error {
	switch command {
	case models.CommandSetMute:
		isMute, err := strconv.ParseBool(detail)
		if err != nil {
			return err
		}
		return tv.SetMute(isMute)
	case models.CommandUp:
		return tv.VolumeUp()
	case models.CommandDown:
		return tv.VolumeDown()
	case models.CommandSet:
		val, err := strconv.Atoi(detail)
		if err != nil {
			return err
		}
		return tv.SetVolume(val)
	case models.CommandAdjust:
		// Get how much we want to adjust by
		adjAmount, err := strconv.Atoi(detail)
		if err != nil {
			return err
		}

		// Get the current volume
		currentVol, err := tv.GetVolume()
		if err != nil {
			return err
		}

		// Set it to the new value
		newVol := currentVol + adjAmount
		if newVol < 0 {
			newVol = 0
		} else if newVol > 100 {
			newVol = 100
		}
		return tv.SetVolume(newVol)
	}

	return ErrCommandUnsupported
}

func getVolumeState(tv *control.LgTv) (models.VolumeState, error) {
	// Get the volume
	vol, err := tv.GetVolume()
	if err != nil {
		return models.VolumeState{}, err
	}

	// Get the mute status
	isMute, err := tv.GetMute()
	if err != nil {
		return models.VolumeState{}, err
	}

	return models.VolumeState{
		Volume:  vol,
		IsMuted: isMute,
	}, nil
}

func handleChannelCommand(tv *control.LgTv, command string, detail string) error {
	switch command {
	case models.CommandUp:
		return tv.ChannelUp()
	case models.CommandDown:
		return tv.ChannelDown()
	case models.CommandSet:
		val, err := strconv.Atoi(detail)
		if err != nil {
			return err
		}
		return tv.SetChannel(val)
	}

	return ErrCommandUnsupported
}

func getChannelState(tv *control.LgTv) (models.ChannelState, error) {
	// Get the current channel
	channel, err := tv.GetCurrentChannel()
	if err != nil {
		return models.ChannelState{}, err
	}

	return models.ChannelState{
		ChannelName:   channel.ChannelName,
		ChannelNumber: channel.ChannelNumber,
	}, nil
}

func handlePowerCommand(tv *control.LgTv, command string, detail string) error {
	switch command {
	case models.CommandOff:
		return tv.TurnOff()
	}

	return ErrCommandUnsupported
}
