const API_VERSION = "latest";

export default {
  // activities
  ACTIVITIES: `/${API_VERSION}/mdmlab/activities`,
  HOST_PAST_ACTIVITIES: (id: number): string => {
    return `/${API_VERSION}/mdmlab/hosts/${id}/activities`;
  },
  HOST_UPCOMING_ACTIVITIES: (id: number): string => {
    return `/${API_VERSION}/mdmlab/hosts/${id}/activities/upcoming`;
  },
  HOST_CANCEL_ACTIVITY: (nodeId: number, uuid: string): string => {
    return `/${API_VERSION}/mdmlab/hosts/${nodeId}/activities/upcoming/${uuid}`;
  },

  CHANGE_PASSWORD: `/${API_VERSION}/mdmlab/change_password`,
  CONFIG: `/${API_VERSION}/mdmlab/config`,
  CONFIRM_EMAIL_CHANGE: (token: string): string => {
    return `/${API_VERSION}/mdmlab/email/change/${token}`;
  },

  DOWNLOAD_INSTALLER: `/${API_VERSION}/mdmlab/download_installer`,
  ENABLE_USER: (id: number): string => {
    return `/${API_VERSION}/mdmlab/users/${id}/enable`;
  },
  FORGOT_PASSWORD: `/${API_VERSION}/mdmlab/forgot_password`,
  GLOBAL_ENROLL_SECRETS: `/${API_VERSION}/mdmlab/spec/enroll_secret`,
  GLOBAL_POLICIES: `/${API_VERSION}/mdmlab/policies`,
  GLOBAL_SCHEDULE: `/${API_VERSION}/mdmlab/schedule`,

  // Device endpoints
  DEVICE_USER_DETAILS: `/${API_VERSION}/mdmlab/device`,
  DEVICE_SOFTWARE: (token: string) =>
    `/${API_VERSION}/mdmlab/device/${token}/software`,
  DEVICE_SOFTWARE_INSTALL: (token: string, softwareTitleId: number) =>
    `/${API_VERSION}/mdmlab/device/${token}/software/install/${softwareTitleId}`,
  DEVICE_USER_MDM_ENROLLMENT_PROFILE: (token: string): string => {
    return `/${API_VERSION}/mdmlab/device/${token}/mdm/apple/manual_enrollment_profile`;
  },
  DEVICE_TRIGGER_LINUX_DISK_ENCRYPTION_KEY_ESCROW: (token: string): string => {
    return `/${API_VERSION}/mdmlab/device/${token}/mdm/linux/trigger_escrow`;
  },

  // Node endpoints
  HOST_SUMMARY: `/${API_VERSION}/mdmlab/host_summary`,
  HOST_QUERY_REPORT: (nodeId: number, queryId: number) =>
    `/${API_VERSION}/mdmlab/hosts/${nodeId}/queries/${queryId}`,
  HOSTS: `/${API_VERSION}/mdmlab/hosts`,
  HOSTS_COUNT: `/${API_VERSION}/mdmlab/hosts/count`,
  HOSTS_DELETE: `/${API_VERSION}/mdmlab/hosts/delete`,
  HOSTS_REPORT: `/${API_VERSION}/mdmlab/hosts/report`,
  HOSTS_TRANSFER: `/${API_VERSION}/mdmlab/hosts/transfer`,
  HOSTS_TRANSFER_BY_FILTER: `/${API_VERSION}/mdmlab/hosts/transfer/filter`,
  HOST_LOCK: (id: number) => `/${API_VERSION}/mdmlab/hosts/${id}/lock`,
  HOST_UNLOCK: (id: number) => `/${API_VERSION}/mdmlab/hosts/${id}/unlock`,
  HOST_WIPE: (id: number) => `/${API_VERSION}/mdmlab/hosts/${id}/wipe`,
  HOST_RESEND_PROFILE: (nodeId: number, profileUUID: string) =>
    `/${API_VERSION}/mdmlab/hosts/${nodeId}/configuration_profiles/${profileUUID}/resend`,
  HOST_SOFTWARE: (id: number) => `/${API_VERSION}/mdmlab/hosts/${id}/software`,
  HOST_SOFTWARE_PACKAGE_INSTALL: (nodeId: number, softwareId: number) =>
    `/${API_VERSION}/mdmlab/hosts/${nodeId}/software/${softwareId}/install`,
  HOST_SOFTWARE_PACKAGE_UNINSTALL: (nodeId: number, softwareId: number) =>
    `/${API_VERSION}/mdmlab/hosts/${nodeId}/software/${softwareId}/uninstall`,

  INVITES: `/${API_VERSION}/mdmlab/invites`,

  // labels
  LABEL: (id: number) => `/${API_VERSION}/mdmlab/labels/${id}`,
  LABELS: `/${API_VERSION}/mdmlab/labels`,
  LABELS_SUMMARY: `/${API_VERSION}/mdmlab/labels/summary`,
  LABEL_HOSTS: (id: number): string => {
    return `/${API_VERSION}/mdmlab/labels/${id}/hosts`;
  },
  LABEL_SPEC_BY_NAME: (labelName: string) => {
    return `/${API_VERSION}/mdmlab/spec/labels/${labelName}`;
  },

  LOGIN: `/${API_VERSION}/mdmlab/login`,
  CREATE_SESSION: `/${API_VERSION}/mdmlab/sessions`,
  LOGOUT: `/${API_VERSION}/mdmlab/logout`,
  MACADMINS: `/${API_VERSION}/mdmlab/macadmins`,

  /**
   * MDM endpoints
   */

  MDM_SUMMARY: `/${API_VERSION}/mdmlab/hosts/summary/mdm`,

  // apple mdm endpoints
  MDM_APPLE: `/${API_VERSION}/mdmlab/mdm/apple`,

  // Apple Business Manager (ABM) endpoints
  MDM_ABM_TOKENS: `/${API_VERSION}/mdmlab/abm_tokens`,
  MDM_ABM_TOKEN: (id: number) => `/${API_VERSION}/mdmlab/abm_tokens/${id}`,
  MDM_ABM_TOKEN_RENEW: (id: number) =>
    `/${API_VERSION}/mdmlab/abm_tokens/${id}/renew`,
  MDM_ABM_TOKEN_TEAMS: (id: number) =>
    `/${API_VERSION}/mdmlab/abm_tokens/${id}/teams`,
  MDM_APPLE_ABM_PUBLIC_KEY: `/${API_VERSION}/mdmlab/mdm/apple/abm_public_key`,
  MDM_APPLE_APNS_CERTIFICATE: `/${API_VERSION}/mdmlab/mdm/apple/apns_certificate`,
  MDM_APPLE_PNS: `/${API_VERSION}/mdmlab/apns`,
  MDM_APPLE_BM: `/${API_VERSION}/mdmlab/abm`, // TODO: Deprecated?
  MDM_APPLE_BM_KEYS: `/${API_VERSION}/mdmlab/mdm/apple/dep/key_pair`,
  MDM_APPLE_VPP_APPS: `/${API_VERSION}/mdmlab/software/app_store_apps`,
  MDM_REQUEST_CSR: `/${API_VERSION}/mdmlab/mdm/apple/request_csr`,

  // Apple VPP endpoints
  MDM_APPLE_VPP_TOKEN: `/${API_VERSION}/mdmlab/mdm/apple/vpp_token`, // TODO: Deprecated?
  MDM_VPP_TOKENS: `/${API_VERSION}/mdmlab/vpp_tokens`,
  MDM_VPP_TOKEN: (id: number) => `/${API_VERSION}/mdmlab/vpp_tokens/${id}`,
  MDM_VPP_TOKENS_RENEW: (id: number) =>
    `/${API_VERSION}/mdmlab/vpp_tokens/${id}/renew`,
  MDM_VPP_TOKEN_TEAMS: (id: number) =>
    `/${API_VERSION}/mdmlab/vpp_tokens/${id}/teams`,

  // MDM profile endpoints
  MDM_PROFILES: `/${API_VERSION}/mdmlab/mdm/profiles`,
  MDM_PROFILE: (id: string) => `/${API_VERSION}/mdmlab/mdm/profiles/${id}`,

  MDM_UPDATE_APPLE_SETTINGS: `/${API_VERSION}/mdmlab/mdm/apple/settings`,
  PROFILES_STATUS_SUMMARY: `/${API_VERSION}/mdmlab/configuration_profiles/summary`,
  DISK_ENCRYPTION: `/${API_VERSION}/mdmlab/disk_encryption`,
  MDM_APPLE_SSO: `/${API_VERSION}/mdmlab/mdm/sso`,
  MDM_APPLE_ENROLLMENT_PROFILE: (token: string, ref?: string) => {
    const query = new URLSearchParams({ token });
    if (ref) {
      query.append("enrollment_reference", ref);
    }
    return `/api/mdm/apple/enroll?${query}`;
  },
  MDM_APPLE_SETUP_ENROLLMENT_PROFILE: `/${API_VERSION}/mdmlab/mdm/apple/enrollment_profile`,
  MDM_BOOTSTRAP_PACKAGE_METADATA: (teamId: number) =>
    `/${API_VERSION}/mdmlab/mdm/bootstrap/${teamId}/metadata`,
  MDM_BOOTSTRAP_PACKAGE: `/${API_VERSION}/mdmlab/mdm/bootstrap`,
  MDM_BOOTSTRAP_PACKAGE_SUMMARY: `/${API_VERSION}/mdmlab/mdm/bootstrap/summary`,
  MDM_SETUP: `/${API_VERSION}/mdmlab/mdm/apple/setup`,
  MDM_EULA: (token: string) => `/${API_VERSION}/mdmlab/mdm/setup/eula/${token}`,
  MDM_EULA_UPLOAD: `/${API_VERSION}/mdmlab/mdm/setup/eula`,
  MDM_EULA_METADATA: `/${API_VERSION}/mdmlab/mdm/setup/eula/metadata`,
  HOST_MDM: (id: number) => `/${API_VERSION}/mdmlab/hosts/${id}/mdm`,
  HOST_MDM_UNENROLL: (id: number) =>
    `/${API_VERSION}/mdmlab/mdm/hosts/${id}/unenroll`,
  HOST_ENCRYPTION_KEY: (id: number) =>
    `/${API_VERSION}/mdmlab/hosts/${id}/encryption_key`,

  ME: `/${API_VERSION}/mdmlab/me`,

  // Disk encryption endpoints
  UPDATE_DISK_ENCRYPTION: `/${API_VERSION}/mdmlab/disk_encryption`,

  // Setup experiece endpoints
  MDM_SETUP_EXPERIENCE: `/${API_VERSION}/mdmlab/setup_experience`,
  MDM_SETUP_EXPERIENCE_SOFTWARE: `/${API_VERSION}/mdmlab/setup_experience/software`,
  MDM_SETUP_EXPERIENCE_SCRIPT: `/${API_VERSION}/mdmlab/setup_experience/script`,

  // OS Version endpoints
  OS_VERSIONS: `/${API_VERSION}/mdmlab/os_versions`,
  OS_VERSION: (id: number) => `/${API_VERSION}/mdmlab/os_versions/${id}`,

  OSQUERY_OPTIONS: `/${API_VERSION}/mdmlab/spec/osquery_options`,
  PACKS: `/${API_VERSION}/mdmlab/packs`,
  PERFORM_REQUIRED_PASSWORD_RESET: `/${API_VERSION}/mdmlab/perform_required_password_reset`,
  QUERIES: `/${API_VERSION}/mdmlab/queries`,
  QUERY_REPORT: (id: number) => `/${API_VERSION}/mdmlab/queries/${id}/report`,
  RESET_PASSWORD: `/${API_VERSION}/mdmlab/reset_password`,
  LIVE_QUERY: `/${API_VERSION}/mdmlab/queries/run`,
  SCHEDULE_QUERY: `/${API_VERSION}/mdmlab/packs/schedule`,
  SCHEDULED_QUERIES: (packId: number): string => {
    return `/${API_VERSION}/mdmlab/packs/${packId}/scheduled`;
  },
  SETUP: `/v1/setup`, // not a typo - hasn't been updated yet

  // Software endpoints
  SOFTWARE: `/${API_VERSION}/mdmlab/software`,
  SOFTWARE_TITLES: `/${API_VERSION}/mdmlab/software/titles`,
  SOFTWARE_TITLE: (id: number) => `/${API_VERSION}/mdmlab/software/titles/${id}`,
  EDIT_SOFTWARE_PACKAGE: (id: number) =>
    `/${API_VERSION}/mdmlab/software/titles/${id}/package`,
  SOFTWARE_VERSIONS: `/${API_VERSION}/mdmlab/software/versions`,
  SOFTWARE_VERSION: (id: number) =>
    `/${API_VERSION}/mdmlab/software/versions/${id}`,
  SOFTWARE_PACKAGE_ADD: `/${API_VERSION}/mdmlab/software/package`,
  SOFTWARE_PACKAGE_TOKEN: (id: number) =>
    `/${API_VERSION}/mdmlab/software/titles/${id}/package/token`,
  SOFTWARE_INSTALL_RESULTS: (uuid: string) =>
    `/${API_VERSION}/mdmlab/software/install/${uuid}/results`,
  SOFTWARE_PACKAGE_INSTALL: (id: number) =>
    `/${API_VERSION}/mdmlab/software/packages/${id}`,
  SOFTWARE_AVAILABLE_FOR_INSTALL: (id: number) =>
    `/${API_VERSION}/mdmlab/software/titles/${id}/available_for_install`,
  SOFTWARE_MDMLAB_MAINTAINED_APPS: `/${API_VERSION}/mdmlab/software/mdmlab_maintained_apps`,
  SOFTWARE_MDMLAB_MAINTAINED_APP: (id: number) =>
    `/${API_VERSION}/mdmlab/software/mdmlab_maintained_apps/${id}`,

  // AI endpoints
  AUTOFILL_POLICY: `/${API_VERSION}/mdmlab/autofill/policy`,

  SSO: `/v1/mdmlab/sso`,
  STATUS_LABEL_COUNTS: `/${API_VERSION}/mdmlab/host_summary`,
  STATUS_LIVE_QUERY: `/${API_VERSION}/mdmlab/status/live_query`,
  STATUS_RESULT_STORE: `/${API_VERSION}/mdmlab/status/result_store`,
  TARGETS: `/${API_VERSION}/mdmlab/targets`,
  TEAM_POLICIES: (teamId: number): string => {
    return `/${API_VERSION}/mdmlab/teams/${teamId}/policies`;
  },
  TEAM_SCHEDULE: (teamId: number): string => {
    return `/${API_VERSION}/mdmlab/teams/${teamId}/schedule`;
  },
  TEAMS: `/${API_VERSION}/mdmlab/teams`,
  TEAMS_AGENT_OPTIONS: (teamId: number): string => {
    return `/${API_VERSION}/mdmlab/teams/${teamId}/agent_options`;
  },
  TEAMS_ENROLL_SECRETS: (teamId: number): string => {
    return `/${API_VERSION}/mdmlab/teams/${teamId}/secrets`;
  },
  TEAM_USERS: (teamId: number): string => {
    return `/${API_VERSION}/mdmlab/teams/${teamId}/users`;
  },
  TEAMS_TRANSFER_HOSTS: (teamId: number): string => {
    return `/${API_VERSION}/mdmlab/teams/${teamId}/hosts`;
  },
  UPDATE_USER_ADMIN: (id: number): string => {
    return `/${API_VERSION}/mdmlab/users/${id}/admin`;
  },
  USER_SESSIONS: (id: number): string => {
    return `/${API_VERSION}/mdmlab/users/${id}/sessions`;
  },
  USERS: `/${API_VERSION}/mdmlab/users`,
  USERS_ADMIN: `/${API_VERSION}/mdmlab/users/admin`,
  VERSION: `/${API_VERSION}/mdmlab/version`,

  // Vulnerabilities endpoints
  VULNERABILITIES: `/${API_VERSION}/mdmlab/vulnerabilities`,
  VULNERABILITY: (cve: string) =>
    `/${API_VERSION}/mdmlab/vulnerabilities/${cve}`,

  // Script endpoints
  HOST_SCRIPTS: (id: number) => `/${API_VERSION}/mdmlab/hosts/${id}/scripts`,
  SCRIPTS: `/${API_VERSION}/mdmlab/scripts`,
  SCRIPT: (id: number) => `/${API_VERSION}/mdmlab/scripts/${id}`,
  SCRIPT_RESULT: (executionId: string) =>
    `/${API_VERSION}/mdmlab/scripts/results/${executionId}`,
  SCRIPT_RUN: `/${API_VERSION}/mdmlab/scripts/run`,

  COMMANDS_RESULTS: `/${API_VERSION}/mdmlab/commands/results`,
};
