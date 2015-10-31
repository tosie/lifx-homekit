package main

import (
  "flag"
  "os"
  "os/signal"
  "math"
  "time"

  log "github.com/Sirupsen/logrus"

  "github.com/pdf/golifx"
  "github.com/pdf/golifx/common"
  "github.com/pdf/golifx/protocol"

  "github.com/brutella/hc/hap"
  "github.com/brutella/hc/model"
  "github.com/brutella/hc/model/accessory"
)

const (
  // from https://github.com/LIFX/LIFXKit/blob/master/LIFXKit/Classes-Common/LFXHSBKColor.h
  HSBKKelvinDefault = uint16(3500)
  HSBKKelvinMin     = uint16(2500)
  HSBKKelvinMax     = uint16(9000)
)

type lightMeta struct {
  transport hap.Transport
  haLight model.LightBulb
  accessory *accessory.Accessory
  light common.Light
  subscription *common.Subscription
}

var (
  client *golifx.Client
  subscription *common.Subscription
  lights map[uint64]lightMeta
  pin string
  debug bool
  timeout int
)

func initClient() error {
  var err error

  client, err = golifx.NewClient(&protocol.V2{Reliable: true})
  if err != nil {
    log.WithField(`error`, err).Error(`Creating LIFX client`)
    return err
  }

    subscription, err = client.NewSubscription()
  if err != nil {
    log.WithField(`error`, err).Error(`Subscribing to events`)
    return err
  }

  events := subscription.Events()
  go func() {
    for {
      select {
      case event := <-events:
        switch event := event.(type) {
        case common.EventNewDevice:
          // TODO: In the future it might not be a light ...
          light, _ := client.GetLightByID(event.Device.ID())
          handleNewLight(light)
        case common.EventExpiredDevice:
          // TODO: In the future it might not be a light ...
          light, _ := client.GetLightByID(event.Device.ID())
          handleExpiredLight(light)
        default:
          log.Debugf("Unhandled event on client: %+v", event)
          continue
        }
      }
    }
  }()

  return nil
}

func handleNewLight(light common.Light) (err error) {
  id := light.ID()

  _, exists := lights[id]
  if exists {
    log.Debugf("A light with the ID '%s' has already been added", id)
    return nil
  }

  subscription, err := light.NewSubscription()
  if err != nil {
    log.WithField(`error`, err).Error(`Subscribing to light events`)
    return err
  }

  label, _ := light.GetLabel()

  log.Infof("Adding light [%s]", label)

  info := model.Info{
    Name: label,
    Manufacturer: "LIFX",
  }

  haLight := accessory.NewLightBulb(info)

  lights[id] = lightMeta{
    light: light,
    subscription: subscription,
    haLight: haLight,
  }
  
  color, _ := light.GetColor()

  // TODO: Convert values regarding their lower and upper limits
  haLight.SetHue(float64(color.Hue) * 360 / math.MaxUint16)
  haLight.SetSaturation(float64(color.Saturation) * 100 / math.MaxUint16)
  haLight.SetBrightness(int(color.Brightness) * 100 / math.MaxUint16)
  
  turnedOn, _ := light.GetPower()
  haLight.SetOn(turnedOn)

  events := subscription.Events()
  go func() {
    for {
      select {
      case event := <-events:
        switch event := event.(type) {
        case common.EventUpdateColor:
          log.Infof("Light: %s, Event: Update Color", id)

          color, _ := light.GetColor()

          // TODO: Convert values regarding their lower and upper limits
          haLight.SetHue(float64(color.Hue) * 360 / math.MaxUint16)
          haLight.SetSaturation(float64(color.Saturation) * 100 / math.MaxUint16)
          haLight.SetBrightness(int(color.Brightness) * 100 / math.MaxUint16)
        case common.EventUpdateLabel:
          // TODO: Find out how to update the name of a homekit device
          log.Infof("Light: %s, Event: Update Label", id)
        case common.EventUpdatePower:
          log.Infof("Light: %s, Event: Update Power", id)

          turnedOn, _ := light.GetPower()
          haLight.SetOn(turnedOn)
        default:
          log.Debugf("Unhandled event on light: %+v", event)
          continue
        }
      }
    }
  }()

  haLight.OnIdentify(func() {
    timeout := 1 * time.Second

    for i := 0; i < 4; i++ {
      toggleLight(light)
      time.Sleep(timeout)
    }
  })

  haLight.OnStateChanged(func(on bool) {
    log.Infof("Setting power of light %s to %s", light.ID(), on)
    light.SetPower(on)
  })

  updateColors := func(light common.Light, haLight model.LightBulb) {
    // HAP: [0...360]
    // LIFX: [0...MAX_UINT16]
    hue := haLight.GetHue()

    // HAP: [0...100]
    // LIFX: [0...MAX_UINT16]
    saturation := haLight.GetSaturation()

    // HAP: [0...100]
    // LIFX: [0...MAX_UINT16]
    brightness := haLight.GetBrightness()

    // TODO: Kelvin [2500..9000]
    
    // TODO: Is this the correct way to convert values from float64 to uint16
    color := common.Color{
      Hue: uint16(math.MaxUint16 * float64(hue) / 360),
      Saturation: uint16(math.MaxUint16 * float64(saturation) / 100),
      Brightness: uint16(math.MaxUint16 * int(brightness) / 100),
      Kelvin: HSBKKelvinDefault,
    }

    light.SetColor(color, 0 * time.Second)
  }

  haLight.OnBrightnessChanged(func(value int) {
    updateColors(light, haLight)
  })

  haLight.OnSaturationChanged(func(value float64) {
    updateColors(light, haLight)
  })

  haLight.OnHueChanged(func(value float64) {
    updateColors(light, haLight)
  })

  transport, err := hap.NewIPTransport(pin, haLight.Accessory)
  if err != nil {
    log.Fatal(err)
  }

  go func() {
    transport.Start()
  }()

  return nil
}

func toggleLight(light common.Light) {
  turnedOn, _ := light.GetPower()
  light.SetPower(!turnedOn)
}

func handleExpiredLight(light common.Light) (err error) {
  id := light.ID()

  meta, exists := lights[id]
  if !exists {
    log.Debugf("Cannot remove a light with the ID '%s' that has not been added before", id)
    return nil
  }

  _ = meta.light.CloseSubscription(lights[id].subscription)

  delete(lights, id)

  return nil
}

// Close closes the LIFX client
func closeClient() {
  for id, _ := range lights {
    handleExpiredLight(lights[id].light)
  }

  _ = client.CloseSubscription(subscription)
  _ = client.Close()

  client = nil
  subscription = nil
}

// Connect establishes a LIFX client and performs device discovery
func startDiscovery() (err error) {
  logger := log.New()

  if debug {
    logger.Level = log.DebugLevel
  }
  golifx.SetLogger(logger)

  if err := initClient(); err != nil {
    tick := time.Tick(2 * time.Second)
    done := make(chan bool)
    select {
    case <-done:
    case <-tick:
      err = initClient()
      if err == nil {
        done <- true
      }
    }
  }

  client.SetDiscoveryInterval(30 * time.Second)

  if timeout > 0 {
    client.SetTimeout(time.Duration(timeout))
  }

  log.Info(`Initiated LIFX client`)

  return nil
}

func waitForInterruption() {
  sig := make(chan os.Signal, 1)

  signal.Notify(sig, os.Interrupt, os.Kill)
  <-sig
}

func main() {
  debug = false
  timeout = 0

  var (
    pinArg = flag.String("pin", "", "PIN used for pairing (must be 8 digits long)")
  )

  flag.Parse()
  pin = *pinArg

  lights = make(map[uint64]lightMeta)

  hap.OnTermination(func() {
    closeClient()
    time.Sleep(100 * time.Millisecond)
    os.Exit(1)
  })

  startDiscovery()

  waitForInterruption()
}
