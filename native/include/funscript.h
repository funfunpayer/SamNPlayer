#pragma once
#include <vector>
#include <cstdint>
#include <string>

namespace samn {

struct Action {
    int64_t at;
    int pos;
};

// MaxActionMs: action times above this (48 h) are treated as corrupt.
constexpr int64_t kMaxActionMs = 48LL * 3600 * 1000;

// CleanActions drops invalid times, clamps pos to 0-100, and sorts by at.
// Mirrors the non-JSON part of funscript.Parse.
std::vector<Action> CleanActions(std::vector<Action> actions);

// WriteActions serializes actions to a minimal .funscript JSON (actions only).
std::string WriteActions(const std::vector<Action>& actions);

}  // namespace samn
