package gpio

import (
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hybridgroup/gobot"
	"github.com/hybridgroup/gobot/gobottest"
)

type DriverAndPinner interface {
	gobot.Driver
	gobot.Pinner
}

type DriverAndEventer interface {
	gobot.Driver
	gobot.Eventer
}

func TestDriverDefaults(t *testing.T) {
	testAdaptor := newGpioTestAdaptor()
	pin := "456"

	drivers := []DriverAndPinner{
		NewGroveTouchDriver(testAdaptor, pin),
		NewGroveSoundSensorDriver(testAdaptor, pin),
		NewGroveButtonDriver(testAdaptor, pin),
		NewGroveBuzzerDriver(testAdaptor, pin),
		NewGroveLightSensorDriver(testAdaptor, pin),
		NewGrovePiezoVibrationSensorDriver(testAdaptor, pin),
		NewGroveLedDriver(testAdaptor, pin),
		NewGroveRotaryDriver(testAdaptor, pin),
		NewGroveRelayDriver(testAdaptor, pin),
		NewGroveMagneticSwitchDriver(testAdaptor, pin),
	}

	for _, driver := range drivers {
		t.Run(getType(driver), func(t *testing.T) {
			gobottest.Assert(t, driver.Connection(), testAdaptor)
			gobottest.Assert(t, driver.Pin(), pin)
			//gobottest.Assert(t, driver.interval, 10*time.Millisecond)
		})
	}
}

func TestDigitalDriverHalt(t *testing.T) {
	testAdaptor := newGpioTestAdaptor()
	pin := "456"

	drivers := []DriverAndEventer{
		NewGroveTouchDriver(testAdaptor, pin),
		NewGroveButtonDriver(testAdaptor, pin),
		NewGroveMagneticSwitchDriver(testAdaptor, pin),
	}

	for _, driver := range drivers {
		t.Run(getType(driver), func(t *testing.T) {

			var callCount int32
			testAdaptorDigitalRead = func() (int, error) {
				atomic.AddInt32(&callCount, 1)
				return 42, nil
			}

			// Start the driver and allow for multiple digital reads
			driver.Start()
			time.Sleep(20 * time.Millisecond)

			driver.Halt()
			lastCallCount := atomic.LoadInt32(&callCount)
			// If driver was not halted, digital reads would still continue
			time.Sleep(20 * time.Millisecond)
			if atomic.LoadInt32(&callCount) != lastCallCount {
				t.Errorf("DigitalRead was called after driver was halted")
			}
		})
	}
}

func TestAnalogDriverHalt(t *testing.T) {
	testAdaptor := newGpioTestAdaptor()
	pin := "456"

	drivers := []DriverAndEventer{
		NewGroveSoundSensorDriver(testAdaptor, pin),
		NewGroveLightSensorDriver(testAdaptor, pin),
		NewGrovePiezoVibrationSensorDriver(testAdaptor, pin),
		NewGroveRotaryDriver(testAdaptor, pin),
	}

	for _, driver := range drivers {
		t.Run(getType(driver), func(t *testing.T) {

			var callCount int32
			testAdaptorAnalogRead = func() (int, error) {
				atomic.AddInt32(&callCount, 1)
				return 42, nil
			}
			// Start the driver and allow for multiple digital reads
			driver.Start()
			time.Sleep(20 * time.Millisecond)

			driver.Halt()
			lastCallCount := atomic.LoadInt32(&callCount)
			// If driver was not halted, digital reads would still continue
			time.Sleep(20 * time.Millisecond)
			if atomic.LoadInt32(&callCount) != lastCallCount {
				t.Errorf("AnalogRead was called after driver was halted")
			}
		})
	}
}

func TestDriverPublishesError(t *testing.T) {
	testAdaptor := newGpioTestAdaptor()
	pin := "456"

	drivers := []DriverAndEventer{
		NewGroveTouchDriver(testAdaptor, pin),
		NewGroveSoundSensorDriver(testAdaptor, pin),
		NewGroveButtonDriver(testAdaptor, pin),
		NewGroveLightSensorDriver(testAdaptor, pin),
		NewGrovePiezoVibrationSensorDriver(testAdaptor, pin),
		NewGroveRotaryDriver(testAdaptor, pin),
		NewGroveMagneticSwitchDriver(testAdaptor, pin),
	}

	for _, driver := range drivers {
		driverType := getType(driver)

		t.Run(driverType, func(t *testing.T) {
			sem := make(chan struct{}, 1)
			// send error
			returnErr := func() (val int, err error) {
				err = errors.New("read error")
				return
			}
			testAdaptorAnalogRead = returnErr
			testAdaptorDigitalRead = returnErr

			gobottest.Assert(t, len(driver.Start()), 0)

			// expect error
			driver.Once(driver.Event(Error), func(data interface{}) {
				gobottest.Assert(t, data.(error).Error(), "read error")
				close(sem)
			})

			select {
			case <-sem:
			case <-time.After(time.Second):
				t.Errorf("%s Event \"Error\" was not published", driverType)
			}
		})

		// Cleanup
		driver.Halt()
	}
}

func getType(driver interface{}) string {
	d := reflect.TypeOf(driver)

	if d.Kind() == reflect.Ptr {
		return d.Elem().Name()
	}

	return d.Name()
}
