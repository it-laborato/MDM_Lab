package service

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/stretchr/testify/require"
)

func TestGetConfig(t *testing.T) {
	t.Run(
		"config cache", func(t *testing.T) {
			oc := OrbitClient{}
			oc.configCache.config = &mdmlab.OrbitConfig{}
			oc.configCache.lastUpdated = time.Now().Add(1 * time.Second)
			config, err := oc.GetConfig()
			require.NoError(t, err)
			require.Equal(t, oc.configCache.config, config)
		},
	)
	t.Run(
		"config cache error", func(t *testing.T) {
			oc := OrbitClient{}
			oc.configCache.config = nil
			oc.configCache.err = errors.New("test error")
			oc.configCache.lastUpdated = time.Now().Add(1 * time.Second)
			config, err := oc.GetConfig()
			require.Error(t, err)
			require.Equal(t, oc.configCache.config, config)
		},
	)
}

func clientWithConfig(cfg *mdmlab.OrbitConfig) *OrbitClient {
	ctx, cancel := context.WithCancel(context.Background())
	oc := &OrbitClient{
		receiverUpdateContext:    ctx,
		receiverUpdateCancelFunc: cancel,
	}
	oc.configCache.config = cfg
	oc.configCache.lastUpdated = time.Now().Add(1 * time.Hour)
	return oc
}

func TestConfigReceiverCalls(t *testing.T) {
	var called1, called2 bool

	testmsg := json.RawMessage("testing")

	rfunc1 := mdmlab.OrbitConfigReceiverFunc(func(cfg *mdmlab.OrbitConfig) error {
		if !reflect.DeepEqual(cfg.Flags, testmsg) {
			return errors.New("not equal testmsg")
		}
		called1 = true
		return nil
	})
	rfunc2 := mdmlab.OrbitConfigReceiverFunc(func(cfg *mdmlab.OrbitConfig) error {
		if !reflect.DeepEqual(cfg.Flags, testmsg) {
			return errors.New("not equal testmsg")
		}
		called2 = true
		return nil
	})

	client := clientWithConfig(&mdmlab.OrbitConfig{Flags: testmsg})
	client.RegisterConfigReceiver(rfunc1)
	client.RegisterConfigReceiver(rfunc2)

	err := client.RunConfigReceivers()
	require.NoError(t, err)

	require.True(t, called1)
	require.True(t, called2)
}

func TestConfigReceiverErrors(t *testing.T) {
	var called1, called2 bool

	rfunc1 := mdmlab.OrbitConfigReceiverFunc(func(cfg *mdmlab.OrbitConfig) error {
		called1 = true
		return nil
	})
	rfunc2 := mdmlab.OrbitConfigReceiverFunc(func(cfg *mdmlab.OrbitConfig) error {
		called2 = true
		return nil
	})
	err1 := errors.New("error1")
	err2 := errors.New("error2")
	efunc1 := mdmlab.OrbitConfigReceiverFunc(func(cfg *mdmlab.OrbitConfig) error {
		return err1
	})
	efunc2 := mdmlab.OrbitConfigReceiverFunc(func(cfg *mdmlab.OrbitConfig) error {
		return err2
	})
	// Make sure we don't get stuck or crash on receiver panic
	pfunc := mdmlab.OrbitConfigReceiverFunc(func(cfg *mdmlab.OrbitConfig) error {
		panic("woah")
	})

	client := clientWithConfig(&mdmlab.OrbitConfig{})
	client.RegisterConfigReceiver(efunc1)
	client.RegisterConfigReceiver(rfunc1)
	client.RegisterConfigReceiver(efunc2)
	client.RegisterConfigReceiver(rfunc2)
	client.RegisterConfigReceiver(pfunc)

	err := client.RunConfigReceivers()
	require.ErrorIs(t, err, err1)
	require.ErrorIs(t, err, err2)

	require.True(t, called1)
	require.True(t, called2)
}

func TestExecuteConfigReceiversCancel(t *testing.T) {
	client := clientWithConfig(&mdmlab.OrbitConfig{})
	client.ReceiverUpdateInterval = 100 * time.Millisecond

	var calls1, calls2 int
	requiredCalls := 4

	cfunc := mdmlab.OrbitConfigReceiverFunc(func(cfg *mdmlab.OrbitConfig) error {
		calls1++
		if calls1 == requiredCalls {
			client.receiverUpdateCancelFunc()
		}
		return nil
	})

	rfunc := mdmlab.OrbitConfigReceiverFunc(func(cfg *mdmlab.OrbitConfig) error {
		calls2++
		return nil
	})

	client.RegisterConfigReceiver(cfunc)
	client.RegisterConfigReceiver(rfunc)

	err := client.ExecuteConfigReceivers()

	require.Nil(t, err)
	require.Equal(t, requiredCalls, calls1)
	require.Equal(t, requiredCalls, calls2)
}

func TestExecuteConfigReceiversInterrupt(t *testing.T) {
	client := clientWithConfig(&mdmlab.OrbitConfig{})
	defer client.receiverUpdateCancelFunc()

	client.ReceiverUpdateInterval = 100 * time.Millisecond

	var called bool
	rfunc := mdmlab.OrbitConfigReceiverFunc(func(cfg *mdmlab.OrbitConfig) error {
		called = true
		return nil
	})

	client.RegisterConfigReceiver(rfunc)

	finChan := make(chan error)
	go func() {
		finChan <- client.ExecuteConfigReceivers()
	}()

	go func() {
		time.Sleep(500 * time.Millisecond)
		client.receiverUpdateCancelFunc()
	}()

	select {
	case err := <-finChan:
		require.Nil(t, err)
		require.True(t, called)
	case <-time.NewTimer(2 * time.Second).C:
		require.Fail(t, "receiver interrupt cancel didn't work")
	}
}
