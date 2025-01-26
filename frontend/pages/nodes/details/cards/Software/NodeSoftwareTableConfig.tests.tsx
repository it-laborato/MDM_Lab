import { createMockNodeSoftwarePackage } from "__mocks__/nodeMock";

import {
  generateActions,
  DEFAULT_ACTION_OPTIONS,
  generateActionsProps,
} from "./NodeSoftwareTableConfig";

describe("generateActions", () => {
  const defaultProps: generateActionsProps = {
    userHasSWWritePermission: true,
    nodeScriptsEnabled: true,
    nodeCanWriteSoftware: true,
    softwareIdActionPending: null,
    softwareId: 1,
    status: null,
    software_package: null,
    app_store_app: null,
    nodeMDMEnrolled: false,
  };

  const defaultPackage = createMockNodeSoftwarePackage();

  it("returns only view details when software does not have software package or app store app", () => {
    const actions = generateActions(defaultProps);
    expect(actions).toEqual([DEFAULT_ACTION_OPTIONS[0]]);
  });

  it("returns default actions for software package when user has write permission and scripts are enabled", () => {
    const actions = generateActions({
      ...defaultProps,
      software_package: defaultPackage,
    });
    expect(actions).toEqual(DEFAULT_ACTION_OPTIONS);
  });

  it("removes install and uninstall actions when user has no write permission", () => {
    const props = {
      ...defaultProps,
      software_package: defaultPackage,
      userHasSWWritePermission: false,
    };
    const actions = generateActions(props);
    expect(actions.find((a) => a.value === "install")).toBeUndefined();
    expect(actions.find((a) => a.value === "uninstall")).toBeUndefined();
  });

  it("disables install and uninstall actions when node scripts are disabled", () => {
    const props = {
      ...defaultProps,
      software_package: defaultPackage,
      nodeScriptsEnabled: false,
    };
    const actions = generateActions(props);
    expect(actions.find((a) => a.value === "install")?.disabled).toBe(true);
    expect(actions.find((a) => a.value === "uninstall")?.disabled).toBe(true);
  });

  it("disables install and uninstall actions when locally pending (waiting for API response)", () => {
    const props = {
      ...defaultProps,
      softwareIdActionPending: 1,
      softwareId: 1,
      software_package: defaultPackage,
    };
    const actions = generateActions(props);
    expect(actions.find((a) => a.value === "install")?.disabled).toBe(true);
    expect(actions.find((a) => a.value === "uninstall")?.disabled).toBe(true);
  });

  it("disables install and uninstall actions when pending install status", () => {
    const props: generateActionsProps = {
      ...defaultProps,
      software_package: defaultPackage,
      status: "pending_install",
    };
    const actions = generateActions(props);
    expect(actions.find((a) => a.value === "install")?.disabled).toBe(true);
    expect(actions.find((a) => a.value === "uninstall")?.disabled).toBe(true);
  });

  it("disables install and uninstall actions when pending uninstall status", () => {
    const props: generateActionsProps = {
      ...defaultProps,
      software_package: defaultPackage,
      status: "pending_uninstall",
    };
    const actions = generateActions(props);
    expect(actions.find((a) => a.value === "install")?.disabled).toBe(true);
    expect(actions.find((a) => a.value === "uninstall")?.disabled).toBe(true);
  });

  it("removes uninstall action for VPP apps", () => {
    const props: generateActionsProps = {
      ...defaultProps,
      app_store_app: {
        app_store_id: "1",
        self_service: false,
        icon_url: "",
        version: "",
        last_install: { command_uuid: "", installed_at: "" },
      },
    };
    const actions = generateActions(props);
    expect(actions.find((a) => a.value === "uninstall")).toBeUndefined();
  });

  it("allows to install VPP apps even if scripts are disabled", () => {
    const props: generateActionsProps = {
      ...defaultProps,
      nodeMDMEnrolled: true,
      nodeScriptsEnabled: false,
      app_store_app: {
        app_store_id: "1",
        self_service: false,
        icon_url: "",
        version: "",
        last_install: { command_uuid: "", installed_at: "" },
      },
    };
    const actions = generateActions(props);
    expect(actions.find((a) => a.value === "install")?.disabled).toBe(false);
    expect(actions.find((a) => a.value === "uninstall")).toBeUndefined();
  });

  it("disables installing VPP app if node is not MDM enrolled", () => {
    const props: generateActionsProps = {
      ...defaultProps,
      nodeScriptsEnabled: false,
      app_store_app: {
        app_store_id: "1",
        self_service: false,
        icon_url: "",
        version: "",
        last_install: { command_uuid: "", installed_at: "" },
      },
    };
    const actions = generateActions(props);
    expect(actions.find((a) => a.value === "install")?.disabled).toBe(true);
    expect(actions.find((a) => a.value === "uninstall")).toBeUndefined();
  });
});
