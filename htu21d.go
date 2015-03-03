package sensors

import (
	"math"
	"time"

	"github.com/davecheney/i2c"
)

const (
	HTU21D_ADDR = 0x40
)

type HTU21D struct {
	RefreshInterval time.Duration
	Temperature     float32
	Humidity        float32

	address uint8
	bus     int
	i2c     *i2c.I2C
	refresh bool
	quit    chan bool
}

func NewHTU21D(address uint8, bus int) (*HTU21D, error) {
	htu := &HTU21D{address: address, bus: bus, RefreshInterval: 10}

	var err error
	htu.i2c, err = i2c.New(htu.address, htu.bus)
	if err != nil {
		return nil, err
	}

	return htu, nil
}

func (this *HTU21D) Close() {
	this.i2c.Close()
}

func (this *HTU21D) StartRefresh() {
	if this.refresh {
		return
	}

	// Read the initial values
	this.Temperature, _ = this.ReadTemperature()
	this.Humidity, _ = this.ReadHumidity()

	go func() {
		this.refresh = true
		for {
			select {
			case <-this.quit:
				return
			default:
				this.Temperature, _ = this.ReadTemperature()
				this.Humidity, _ = this.ReadHumidity()

				time.Sleep(this.RefreshInterval * time.Second)
			}
		}
		this.refresh = false
	}()
}

func (this *HTU21D) StopRefresh() {
	this.quit <- true
}

func (this *HTU21D) ReadTemperature() (float32, error) {
	res := make([]byte, 3)
	_, err := this.i2c.WriteByte(0xF3)
	if err != nil {
		return 0, err
	}
	time.Sleep(100 * time.Millisecond)
	_, err = this.i2c.Read(res)
	if err != nil {
		return 0, err
	}

	var raw_temp uint16
	raw_temp = uint16(res[0])*uint16(256) + uint16(res[1])
	// Reset 2 status bits
	raw_temp = raw_temp & 0xFFFC
	temp := -46.85 + (175.72 * (float32(raw_temp) / float32(math.Pow(2, 16))))

	return temp, nil
}

func (this *HTU21D) ReadHumidity() (float32, error) {
	res := make([]byte, 3)
	_, err := this.i2c.WriteByte(0xF5)
	if err != nil {
		return 0, err
	}
	time.Sleep(100 * time.Millisecond)
	_, err = this.i2c.Read(res)
	if err != nil {
		return 0, err
	}

	var raw_humidity uint16
	raw_humidity = uint16(res[0])*uint16(256) + uint16(res[1])
	// Reset 2 status bits
	raw_humidity = raw_humidity & 0xFFFC
	humidity := -6 + (125 * (float32(raw_humidity) / float32(math.Pow(2, 16))))

	return humidity, nil
}
