#ifndef RUUVI_MEASUREMENT_
#define RUUVI_MEASUREMENT_

// Partly from Measurement.cpp - Part of ESP32 Ruuvitag Collector
// https://github.com/hpirila/ESP32-Ruuvitag-Collector/blob/master/src/Measurement.cpp

#include <vector>
#include <string>
#include <sstream>

namespace Measurement
{
  enum MeasurementType
  {
    undefined,
    ruuviV3,
    ruuviV5
  };

  struct RuuviMeasurement
  {
    std::string mac;
    double temperature{0.0};
    double humidity{0.0};
    double pressure{0.0};
    time_t epoch{0};
    int accelX{0};
    int accelY{0};
    int accelZ{0};
    int voltage{0};
    int power{0};
    int moveCount{0};
    int sequence{0};
  };

  int getShort(std::vector<uint8_t> data, int index)
  {
    return (short)((data[index] << 8) + (data[index + 1]));
  }

  int getShortone(std::vector<uint8_t> data, int index)
  {
    return (short)((data[index]));
  }

  unsigned int getUShort(std::vector<uint8_t> data, int index)
  {
    return (unsigned short)((data[index] << 8) + (data[index + 1]));
  }

  unsigned int getUShortone(std::vector<uint8_t> data, int index)
  {
    return (unsigned short)((data[index]));
  }

  // Function to read and process the data from the device
  RuuviMeasurement readDataFromDevice(NimBLEAdvertisedDevice device)
  {
    std::string data = device.getManufacturerData();
    if (data.length() > 2)
    {
      RuuviMeasurement m = parseData(data, device.getAddress().toString(), measurementTimeNow);
      return m;
    }
    else
    {
      return RuuviMeasurement();
    }
  }

  RuuviMeasurement parseData(const std::string &dataIn, const std::string &mac, const time_t &epoch)
  {
    int measurementType = MeasurementType::undefined;
    if (dataIn[0] == 0x99 && dataIn[1] == 0x04)
    {
      if (dataIn[2] == 0x3 && dataIn.length() > 15)
      {
        measurementType = MeasurementType::ruuviV3;
      }
      if (dataIn[2] == 0x5 && dataIn.length() > 19)
      {
        measurementType = MeasurementType::ruuviV5;
      }
    }

    std::vector<uint8_t> data(dataIn.begin(), dataIn.end());
    RuuviMeasurement m;
    m.mac = mac;
    m.epoch = epoch;
    switch (measurementType)
    {
    case MeasurementType::ruuviV3:
      m.temperature = (double)(getUShortone(data, 4) & 0b01111111) + (double)getUShortone(data, 5) / 100;
      m.temperature = (getUShortone(data, 4) & 0b10000000) == 128 ? m.temperature * -1 : m.temperature;
      m.humidity = (double)getUShortone(data, 3) * 0.5;
      m.pressure = (double)getUShort(data, 6) / 100 + 500;
      m.accelX = getShort(data, 8);
      m.accelY = getShort(data, 10);
      m.accelZ = getShort(data, 12);
      m.voltage = (short)getUShort(data, 14);
      break;
    case MeasurementType::ruuviV5:
      m.temperature = (double)getShort(data, 3) * 0.005;
      m.humidity = (double)getUShort(data, 5) * 0.0025;
      m.pressure = (double)getUShort(data, 7) / 100 + 500;
      m.accelX = getShort(data, 9);
      m.accelY = getShort(data, 11);
      m.accelZ = getShort(data, 13);
      m.voltage = (data[15] << 3 | data[16] >> 5) + 1600;
      m.power = (data[16] & 0x1F) * 2 - 40;
      m.moveCount = getUShortone(data, 17);
      m.sequence = getUShort(data, 18);
      break;
    default:
      break;
    }
    return m;
  }

  std::string measurementToJsonString(const RuuviMeasurement &m, bool short_field_names = false)
  {
    std::stringstream ss;
    ss << "{";

    // if these values present, then probably sensor does not supply these measurementes i.e. it does not have sensors for these measurements
    // Lets not send these. Or let data consumer deal with these
    // double maxPressure = 1155.35;
    // double maxHumidity = 163.838;

    if (short_field_names)
    {
      ss << "\"a\": " << "\"" << m.mac << "\"" << ", " << "\"t\": " << m.temperature << ", " << "\"p\": " << m.pressure << ", " << "\"h\": " << m.humidity << ", " << "\"x\": " << m.accelX << ", " << "\"y\": " << m.accelY << ", " << "\"z\": " << m.accelZ << ", " << "\"b\": " << m.voltage << ", " << "\"e\": " << m.epoch << ", " << "\"l\": " << m.power << ", " << "\"m\": " << m.moveCount << ", " << "\"s\": " << m.sequence;
    }
    else
    {
      ss << "\"mac\": " << "\"" << m.mac << "\"" << ", " << "\"temperature\": " << m.temperature << ", " << "\"pressure\": " << m.pressure << ", " << "\"humidity\": " << m.humidity << ", " << "\"accelX\": " << m.accelX << ", " << "\"accelY\": " << m.accelY << ", " << "\"accelZ\": " << m.accelZ << ", " << "\"battery\": " << m.voltage << ", " << "\"epoch\": " << m.epoch << ", " << "\"txdbm\": " << m.power << ", " << "\"move\": " << m.moveCount << ", " << "\"sequence\": " << m.sequence;
    }
    ss << "}";

    return ss.str();
  }
}

#endif