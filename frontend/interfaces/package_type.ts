const mdmlabMaintainedPackageTypes = ["dmg", "zip"] as const;
const unixPackageTypes = ["pkg", "deb", "rpm", "dmg", "zip"] as const;
const windowsPackageTypes = ["msi", "exe"] as const;
export const packageTypes = [
  ...unixPackageTypes,
  ...windowsPackageTypes,
] as const;

export type WindowsPackageType = typeof windowsPackageTypes[number];
export type UnixPackageType = typeof unixPackageTypes[number];
export type MdmlabMaintainedPackageType = typeof mdmlabMaintainedPackageTypes[number];
export type PackageType =
  | WindowsPackageType
  | UnixPackageType
  | MdmlabMaintainedPackageType;

export const isWindowsPackageType = (s: any): s is WindowsPackageType => {
  return windowsPackageTypes.includes(s);
};

export const isUnixPackageType = (s: any): s is UnixPackageType => {
  return unixPackageTypes.includes(s);
};

export const isMdmlabMaintainedPackageType = (
  s: any
): s is MdmlabMaintainedPackageType => {
  return mdmlabMaintainedPackageTypes.includes(s);
};

export const isPackageType = (s: any): s is PackageType => {
  return packageTypes.includes(s);
};
