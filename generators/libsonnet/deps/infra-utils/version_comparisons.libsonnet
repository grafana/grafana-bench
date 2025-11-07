{
  local root = self,
  local _split(version) = [
    std.parseInt(c)
    for c in std.split(version, '.')
  ],
  util:: {
    getMajorVersion(version):: (
      local split = _split(version);
      split[0]
    ),
    getMajorMinorVersion(version):: (
      local split = _split(version);
      // Pad with trailing 0's when getting only a MAJOR
      if std.length(split) == 1 then
        [split[0], 0]
      else
        split[0:2]
    ),
    getMajorMinorPatchVersion(version):: (
      local split = _split(version);
      // Pad with trailing 0's when getting only a MAJOR or MAJOR.MINOR
      if std.length(split) == 1 then
        [split[0], 0, 0]
      else if std.length(split) == 2 then
        [split[0], split[1], 0]
      else
        split[0:3]
    ),
  },
  major:: {
    equal(a, b):: (
      local aMajor = root.util.getMajorVersion(a);
      local bMajor = root.util.getMajorVersion(b);
      aMajor == bMajor
    ),
    greaterThan(a, b):: (
      local aMajor = root.util.getMajorVersion(a);
      local bMajor = root.util.getMajorVersion(b);
      aMajor > bMajor
    ),
    greaterOrEqualThan(a, b):: self.greaterThan(a, b) || self.equal(a, b),
    lessThan(a, b):: self.greaterThan(b, a),
  },
  majorMinor:: {
    equal(a, b):: (
      local aMajorMinor = root.util.getMajorMinorVersion(a);
      local bMajorMinor = root.util.getMajorMinorVersion(b);
      (aMajorMinor[0] == bMajorMinor[0])
      && (aMajorMinor[1] == bMajorMinor[1])
    ),
    greaterThan(a, b):: (
      local aMajorMinor = root.util.getMajorMinorVersion(a);
      local bMajorMinor = root.util.getMajorMinorVersion(b);

      // Need to check >= on the first N-1 comparisons, because if they are equal
      // we want to fallback to the next one.
      (
        aMajorMinor[0] > bMajorMinor[0]
      )
      || (
        (aMajorMinor[0] == bMajorMinor[0])
        && (aMajorMinor[1] > bMajorMinor[1])
      )
    ),
    greaterOrEqualThan(a, b):: self.greaterThan(a, b) || self.equal(a, b),
    lessThan(a, b):: self.greaterThan(b, a),
  },
  majorMinorPatch:: {
    equal(a, b):: (
      local aMajorMinorPatch = root.util.getMajorMinorPatchVersion(a);
      local bMajorMinorPatch = root.util.getMajorMinorPatchVersion(b);
      (aMajorMinorPatch[0] == bMajorMinorPatch[0])
      && (aMajorMinorPatch[1] == bMajorMinorPatch[1])
      && (aMajorMinorPatch[2] == bMajorMinorPatch[2])
    ),
    greaterThan(a, b):: (
      local aMajorMinorPatch = root.util.getMajorMinorPatchVersion(a);
      local bMajorMinorPatch = root.util.getMajorMinorPatchVersion(b);
      // Need to check >= on the first N-1 comparisons, because if they are equal
      // we want to fallback to the next one.
      (
        aMajorMinorPatch[0] > bMajorMinorPatch[0]
      )
      || (
        (aMajorMinorPatch[0] == bMajorMinorPatch[0])
        && (aMajorMinorPatch[1] > bMajorMinorPatch[1])
      )
      || (
        (aMajorMinorPatch[0] == bMajorMinorPatch[0])
        && (aMajorMinorPatch[1] == bMajorMinorPatch[1])
        && (aMajorMinorPatch[2] > bMajorMinorPatch[2])
      )
    ),
    greaterOrEqualThan(a, b):: self.greaterThan(a, b) || self.equal(a, b),
    lessThan(a, b):: self.greaterThan(b, a),
  },
}
