import { IPolicyNew } from "interfaces/policy";
import { CommaSeparatedPlatformString } from "interfaces/platform";

const DEFAULT_POLICY_PLATFORM: CommaSeparatedPlatformString = "";

export const DEFAULT_POLICY = {
  id: 1,
  name: "Is osquery running?",
  query: "SELECT 1 FROM osquery_info WHERE start_time > 1;",
  description: "Checks if the osquery process has started on the node.",
  author_id: 42,
  author_name: "John",
  author_email: "john@example.com",
  resolution: "Resolution steps",
  platform: DEFAULT_POLICY_PLATFORM,
  passing_node_count: 2000,
  failing_node_count: 300,
  created_at: "",
  updated_at: "",
  critical: false,
};

// We disable some linting and prettier for DEFAULT_POLICIES object because we
// need to keep some backslash(\) characters in some of the query string values.

/* eslint-disable no-useless-escape */
// prettier-ignore
export const DEFAULT_POLICIES: IPolicyNew[] = [
   {
    key: 3,
    query:
      "SELECT 1 from windows_security_center wsc CROSS JOIN windows_security_products wsp WHERE antivirus = 'Good' AND type = 'Antivirus' AND signatures_up_to_date=1;",
    name: "Antivirus healthy (Windows)",
    description:
      "Lack of active, updated antivirus exposes the workstation to malware and security threats.",
    resolution:
      "Ensure Windows Defender or your third-party antivirus is running, up to date, and visible in the Windows Security Center.",
    critical: false,
    platform: "windows",
  },
  {
    key: 7,
    query:
      "SELECT 1 FROM bitlocker_info WHERE drive_letter='C:' AND protection_status=1;",
    name: "Full disk encryption enabled (Windows)",
    description:
      "If BitLocker is disabled, the workstation's data is at risk of unauthorized access and theft.",
    resolution:
      "Full disk encryption will be enabled to secure data.",
    critical: false,
    platform: "windows",
  },
  {
    key: 14,
    query:
      "SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\\Software\\Microsoft\\Windows\\CurrentVersion\\Policies\\System\\InactivityTimeoutSecs' AND CAST(data as INTEGER) <= 1800;",
    name: "Screen lock enabled (Windows)",
    description:
      "Devices with inactive timeout settings over 30 minutes risk prolonged unauthorized access if left unattended, exposing sensitive data.",
    resolution:
      "Enable the Interactive Logon: Machine inactivity limit setting with a value of 1800 seconds or lower.",
    critical: false,
    platform: "windows",
  },
  {
    key: 31,
    query:
      "SELECT 1 FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\SharedAccess\\Parameters\\FirewallPolicy\\DomainProfile\\EnableFirewall' AND CAST(data as integer) = 1;",
    name: "Windows Firewall, domain profile enabled (Windows)",
    description:
      "If the Windows Firewall is not enabled for the domain profile, the workstation may be more vulnerable to unauthorized network access and potential security breaches.",
    resolution:
      "The Windows Firewall will be enabled for the domain profile.",
    critical: false,
    platform: "windows",
  },
  {
    key: 32,
    query:
      "SELECT 1 FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\SharedAccess\\Parameters\\FirewallPolicy\\StandardProfile\\EnableFirewall' AND CAST(data as integer) = 1;",
    name: "Windows Firewall, private profile enabled (Windows)",
    description:
      "If the Windows Firewall is not enabled for the private profile, the workstation may be more susceptible to unauthorized access and potential security breaches, particularly when connected to private networks.",
    resolution:
      "The Windows Firewall will be enabled for the private profile",
    critical: false,
    platform: "windows",
  },
  {
    key: 33,
    query:
      "SELECT 1 FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\SharedAccess\\Parameters\\FirewallPolicy\\PublicProfile\\EnableFirewall' AND CAST(data as integer) = 1;",
    name: "Windows Firewall, public profile enabled (Windows)",
    description:
      "If the Windows Firewall is not enabled for the public profile, the workstation may be more vulnerable to unauthorized access and potential security threats, especially when connected to public networks.",
    resolution:
      "The Windows Firewall will be enabled for the public profile.",
    critical: false,
    platform: "windows",
  },
  {
    key: 34,
    query:
      "SELECT 1 FROM windows_optional_features WHERE name = 'SMB1Protocol-Client' AND state != 1;",
    name: "SMBv1 client driver disabled (Windows)",
    description: "Leaving the SMBv1 client enabled increases vulnerability to security threats and potential exploitation by malicious actors.",
    resolution:
      "The SMBv1 client will be disabled.",
    critical: false,
    platform: "windows",
  },
  {
    key: 35,
    query:
      "SELECT 1 FROM windows_optional_features WHERE name = 'SMB1Protocol-Server' AND state != 1",
    name: "SMBv1 server disabled (Windows)",
    description: "Leaving the SMBv1 server enabled exposes the workstation to potential security vulnerabilities and exploitation by malicious actors.",
    resolution:
      "The SMBv1 server will be disabled.",
    critical: false,
    platform: "windows",
  },
  {
    key: 36,
    query:
      "SELECT 1 FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINE\\Software\\Policies\\Microsoft\\Windows NT\\DNSClient\\EnableMulticast' AND CAST(data as integer) = 0;",
    name: "LLMNR disabled (Windows)",
    description:
      "If the workstation does not have LLMNR disabled, it could be vulnerable to DNS spoofing attacks, potentially leading to unauthorized access or data interception.",
    resolution:
      "LLMNR will be disabled on your system.",
    critical: false,
    platform: "windows",
  },
  {
    key: 37,
    query:
      "SELECT 1 FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINE\\Software\\Policies\\Microsoft\\Windows\\Windows\\Update\\AU\\NoAutoUpdate' AND CAST(data as integer) = 0;",
    name: "Automatic updates enabled (Windows)",
    description:
      "Enabling automatic updates ensures the computer downloads and installs security and other important updates automatically.",
    resolution:
      "Automatic updates will be enabled.",
    critical: false,
    platform: "windows",
  },
];
