#include <iostream>
#include <mutex>
#include <sstream>
#include <string>
#include <thread>

#include "log.h"

#include "util_string.h"

static std::mutex log_mutex;

void dxvk::log(const std::string& prefix, const std::string& message) {
  std::lock_guard<std::mutex> lock(log_mutex);

  std::stringstream stream(message);
  std::string       line;

  while (std::getline(stream, line, '\n')) {
    std::cerr << prefix << ": " << line << std::endl;
  }
}
