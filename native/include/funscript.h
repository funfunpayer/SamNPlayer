#pragma once
#include <vector>
#include <string>
#include <cstdint>
#include <optional>

namespace samn {

struct Action {
    int64_t at = 0;
    int pos = 0;
};

struct Script {
    std::vector<Action> actions;
    bool inverted = false;
    int dropped = 0;
    int64_t duration_meta = 0;  // ms, after OFS scaling rules
    std::string metadata_json;  // full metadata object (duration adjusted), empty if none
};

// MaxActionMs: action times above this (48 h) are treated as corrupt.
constexpr int64_t MaxActionMs = 48LL * 3600 * 1000;

// Parse a funscript JSON string. Throws std::runtime_error on hard failures
// (invalid JSON, no actions, all times corrupt). Soft drops corrupt times.
// Preserves full metadata (with duration OFS rules applied).
Script Parse(const std::string& json_str);

// Write a funscript JSON (actions + inverted + full metadata if present).
std::string Write(const Script& s);

}  // namespace samn
