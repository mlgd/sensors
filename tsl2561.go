package sensors

import (
	"math"
	"time"

	"github.com/davecheney/i2c"
)

const (
	TSL2561_ADDR   = 0x39
	TSL2561_ADDR_1 = 0x49
	TSL2561_ADDR_0 = 0x29
)

type TSL2561 struct {
	RefreshInterval time.Duration
	Lux             float32

	address uint8 // default : 0x39
	bus     int
	i2c     *i2c.I2C
	gain    int
	refresh bool
	quit    chan bool
}

func NewTSL2561(address uint8, bus int) (*TSL2561, error) {
	tsl := &TSL2561{address: address, bus: bus, RefreshInterval: 10}

	var err error
	tsl.i2c, err = i2c.New(tsl.address, tsl.bus)
	if err != nil {
		return nil, err
	}

	return tsl, nil
}

func (this *TSL2561) Close() {
	this.i2c.Close()
}

func (this *TSL2561) StartRefresh() {
	if this.refresh {
		return
	}

	// Read the initial value
	this.Lux, _ = this.ReadLux()

	go func() {
		this.refresh = true
		for {
			select {
			case <-this.quit:
				return
			default:
				this.Lux, _ = this.ReadLux()

				time.Sleep(this.RefreshInterval * time.Second)
			}
		}
		this.refresh = false
	}()
}

func (this *TSL2561) StopRefresh() {
	this.quit <- true
}

func (this *TSL2561) Gain() int {
	return this.gain
}

func (this *TSL2561) SetGain(gain int) {
	if this.gain != gain {
		if gain == 1 {
			command := []byte{0x81, 0x02}
			this.i2c.Write(command)
		} else {
			command := []byte{0x81, 0x12}
			this.i2c.Write(command)
		}
		this.gain = gain
		time.Sleep(800 * time.Millisecond)
	}
}

func (this *TSL2561) readFull() (uint16, error) {
	res := make([]byte, 2)
	_, err := this.i2c.WriteByte(0xAC)
	if err != nil {
		return 0, err
	}
	_, err = this.i2c.Read(res)
	if err != nil {
		return 0, err
	}
	full := uint16(res[1])*256 + uint16(res[0])
	return full, nil
}

func (this *TSL2561) readIr() (uint16, error) {
	res := make([]byte, 2)
	_, err := this.i2c.WriteByte(0xAE)
	if err != nil {
		return 0, err
	}
	_, err = this.i2c.Read(res)
	if err != nil {
		return 0, err
	}
	ir := uint16(res[1])*256 + uint16(res[0])
	return ir, nil
}

func (this *TSL2561) ReadLux() (float32, error) {
	var ambient uint16
	var ir uint16
	var ratio float32
	var err error

	switch {
	case this.gain == 1 || this.gain == 16:
		ambient, err = this.readFull()
		if err != nil {
			return 0, err
		}

		ir, err = this.readIr()
		if err != nil {
			return 0, err
		}
	case this.gain == 0:
		this.gain = 16

		ambient, err = this.readFull()
		if err != nil {
			return 0, err
		}
		if ambient < 0xFFFF {
			ir, err = this.readIr()
			if err != nil {
				return 0, err
			}
		}
		if ambient >= 0xFFFF || ir >= 0xFFFF {
			this.gain = 1

			ambient, err = this.readFull()
			if err != nil {
				return 0, err
			}

			ir, err = this.readIr()
			if err != nil {
				return 0, err
			}
		}
	}

	if this.gain == 1 {
		ambient *= 16
		ir *= 16
	}

	ratio = float32(ir) / float32(ambient)

	var lux float32
	if ratio >= 0 && ratio <= 0.52 {
		lux = (0.0315 * float32(ambient)) - (0.0593 * float32(ambient) * float32(math.Pow(float64(ratio), 1.4)))
	} else if ratio <= 0.65 {
		lux = (0.0229 * float32(ambient)) - (0.0291 * float32(ir))
	} else if ratio <= 0.80 {
		lux = (0.0157 * float32(ambient)) - (0.018 * float32(ir))
	} else if ratio <= 1.3 {
		lux = (0.00338 * float32(ambient)) - (0.0026 * float32(ir))
	} else if ratio > 1.3 {
		lux = 0
	}

	return lux, nil
}
