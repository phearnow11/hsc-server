package utils

import (
	"log"
	"os"
	"strings"
)

// OLD CODE --- used by ver1
func Data_convert_up(data_units []string, convertion_unit float64, data float64, base_unit_index int) (float64, string, string) {
	short_unit_name := ""
	log.Println("Preparing to convert data:", data, data_units[base_unit_index])
	for data > 100 {
		data /= float64(convertion_unit)
		if data_units[base_unit_index] == "millisecond" {
			convertion_unit = 60
		}
		base_unit_index++
	}
	final_unit_name := data_units[base_unit_index]
	short_unit_name += string(final_unit_name[0])
	if data_units[0] == "byte" {
		short_unit_name = strings.ToUpper(short_unit_name)
		short_unit_name += "b"
	}

	log.Println("Data converted:", data, short_unit_name)
	return data, final_unit_name, short_unit_name
}

func Data_convert_down(data_units []string, convertion_unit float64, data float64, base_unit_index int) (float64, string, string) {
	if base_unit_index == 0 {
		log.Println("ERROR AT CONVERTION DOWN: UNIT NOT SUPPORTED")
	}
	short_unit_name := ""
	log.Println("Preparing to convert data:", data, data_units[base_unit_index])
	for data < 1 {
		if base_unit_index == 0 {
			log.Println("UNIT NOT SUPPORTED, STOP CONVERTION PROCESS")
			break
		}
		data *= convertion_unit
		if data_units[base_unit_index-1] == "millisecond" {
			convertion_unit = 1000
		}
		base_unit_index--
	}
	final_unit_name := data_units[base_unit_index]
	short_unit_name += string(final_unit_name[0])
	if data_units[0] == "byte" {
		short_unit_name = strings.ToUpper(short_unit_name)
		short_unit_name += "b"
	}

	log.Println("Data converted:", data, short_unit_name)
	return data, final_unit_name, short_unit_name
}

// LATEST
func Contain(arr []string, target string) int {
	for i, val := range arr {
		if target == val {
			return i
		}
	}
	return -1
}

var (
	BYTE_FAMILY = []string{"byte", "kilobyte", "megabyte", "gigabyte", "terabyte", ""}
	TIME_FAMILY = []string{"nanosecond", "millisecond", "second", "minute", "hour", ""}
	// NOTE: ONLY SUPPORT UNIT WITH 2 RATE CONVERTION
	SUPPORTED_UNIT_FAMILY_DATA             = [][]string{BYTE_FAMILY, TIME_FAMILY}
	SUPPORTED_UNIT_FAMILY_NAME             = []string{"bytes", "time"}
	SUPPORTED_UNIT_FAMILY_RATE             = [][]float64{{1024}, {1000, 60}}
	SUPPORTED_UNIT_FAMILY_RATE_INTERCHANGE = []uint{0, 2}
)

func Get_supported_unit_family_and_convertion_rate(unit_family string, unit_name string) ([]string, float64, int) {
	if supported_index := Contain(SUPPORTED_UNIT_FAMILY_NAME, unit_family); supported_index >= 0 {
		unit_index := Contain(SUPPORTED_UNIT_FAMILY_DATA[supported_index], unit_name)
		if unit_index < 0 {
			log.Println("Unit is not SUPPORTED: UNIT NAME NOT FOUND")
			return []string{}, 0, -1
		}
		if SUPPORTED_UNIT_FAMILY_RATE_INTERCHANGE[supported_index] > 0 {
			if unit_index < int(SUPPORTED_UNIT_FAMILY_RATE_INTERCHANGE[supported_index]) {
				return SUPPORTED_UNIT_FAMILY_DATA[supported_index], SUPPORTED_UNIT_FAMILY_RATE[supported_index][0], unit_index
			} else {
				return SUPPORTED_UNIT_FAMILY_DATA[supported_index], SUPPORTED_UNIT_FAMILY_RATE[supported_index][1], unit_index
			}
		}
		return SUPPORTED_UNIT_FAMILY_DATA[supported_index], SUPPORTED_UNIT_FAMILY_RATE[supported_index][0], unit_index
	}
	log.Println("Unit is not SUPPORTED: NOT SUPPORTED")
	return []string{}, 0.0, -1
}

func short_unit_name_extraction(data_units []string, index uint) string {
	res := string(data_units[index][0])
	if Contain(data_units, "byte") >= 0 {
		res = strings.ToUpper(res)
		res += "b"
	} else if second_index := Contain(data_units, "second"); second_index >= 0 {
		if index < uint(second_index) {
			res += "s"
		}
	} else {
		log.Println("Unit is not SUPPORTED")
		return ""
	}
	return res
}

func Data_convertion(data_units []string, convertion_rate float64, data float64, base_unit_index int) (float64, string, string) {
	if data >= 1 && data < 1000 {
		log.Println("DATACONVERTION: Convertion Successfully!:", data)
		return data, data_units[base_unit_index], short_unit_name_extraction(data_units, uint(base_unit_index))
	}
	if base_unit_index < 0 {
		log.Println("DATACONVERTION ERROR: BASE UNIT INDEX LESS THAN 0")
		return data, data_units[base_unit_index], short_unit_name_extraction(data_units, uint(base_unit_index))
	}
	if data < 1 {
		if data_units[base_unit_index] == "second" {
			convertion_rate = 1000
		}
		base_unit_index -= 1
	} else if data >= 1000 {
		if data_units[base_unit_index] == "second" {
			convertion_rate = 60
		}
		convertion_rate = 1 / convertion_rate
		base_unit_index += 1
	}
	if base_unit_index < 0 {
		log.Println("DATACONVERTION ERROR: BASE UNIT INDEX LESS THAN 0 AFTER CONVERTION")
		return data, data_units[base_unit_index], short_unit_name_extraction(data_units, uint(base_unit_index))
	}
	return Data_convertion(data_units, convertion_rate, data*convertion_rate, base_unit_index)
}

func CreateFile(name string) error {
	err := os.MkdirAll(name, 0755)
	if err != nil {
		return err
	}
	return nil
}
