package hardware

// I2C Constants.
const (
	// Addresses.
	displayAddress       uint16 = 0x3C
	daughterboardAddress uint16 = 0x1A

	// Command bytes.
	cmdEnableDisplay        byte = 0xAF
	cmdDisableDisplay       byte = 0xAE
	cmdDisplayRAMContent    byte = 0xA4
	cmdEntireDisplayOn      byte = 0xA5
	cmdSetBrightness        byte = 0x81
	cmdResetBrightness      byte = 0x7F
	cmdNormalDisplay        byte = 0xA6
	cmdInvertedDisplay      byte = 0xA7
	cmdSetMemoryMode        byte = 0x20
	cmdSetColumnAddress     byte = 0x21
	cmdSetPageAddress       byte = 0x22
	cmdPageStartAddressBase byte = 0xB0
	cmdHorizontalOffsetBase byte = 0x40
	cmdSystemHalt           byte = 0xFF
	cmdWriteData            byte = 0x40

	// Misc.
	displayWidth  int = 128
	displayHeight int = 64
)

// GPIO Constants.
const (
	buttonPinAddress byte = 0x04 //nolint:unused
)
