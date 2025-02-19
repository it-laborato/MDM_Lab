import React from "react";
import { screen } from "@testing-library/react";

import { IDeviceUserResponse, INodeDevice } from "interfaces/node";
import createMockNode from "__mocks__/nodeMock";
import mockServer from "test/mock-server";
import { createCustomRenderer } from "test/test-utils";
import { customDeviceHandler } from "test/handlers/device-handler";
import DeviceUserPage from "./DeviceUserPage";

const mockRouter = {
  push: jest.fn(),
  replace: jest.fn(),
  goBack: jest.fn(),
  goForward: jest.fn(),
  go: jest.fn(),
  setRouteLeaveHook: jest.fn(),
  isActive: jest.fn(),
  createHref: jest.fn(),
  createPath: jest.fn(),
};

const mockLocation = {
  pathname: "",
  query: {
    vulnerable: undefined,
    page: undefined,
    query: undefined,
    order_key: undefined,
    order_direction: undefined,
  },
  search: undefined,
};

describe("Device User Page", () => {
  it("hides the software tab if the device has no software", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });

    // TODO: fix return type from render
    const { user } = render(
      <DeviceUserPage
        router={mockRouter}
        params={{ device_auth_token: "testToken" }}
        location={mockLocation}
      />
    );

    // waiting for the device data to render
    await screen.findByText("About");

    expect(screen.queryByText(/Software/)).not.toBeInTheDocument();

    // TODO: Fix this to the new copy
    // expect(screen.getByText("No software detected")).toBeInTheDocument();
  });

  describe("MDM enrollment", () => {
    const setupTest = async (overrides: Partial<IDeviceUserResponse>) => {
      mockServer.use(customDeviceHandler(overrides));

      const render = createCustomRenderer({
        withBackendMock: true,
      });

      const { user } = await render(
        <DeviceUserPage
          router={mockRouter}
          params={{ device_auth_token: "testToken" }}
          location={mockLocation}
        />
      );

      // waiting for the device data to render
      await screen.findByText("About");

      return user;
    };

    it("shows a banner when MDM is configured and the device is unenrolled", async () => {
      const node = createMockNode() as INodeDevice;
      node.mdm.enrollment_status = "Off";
      node.platform = "darwin";
      node.dep_assigned_to_mdmlab = false;

      const user = await setupTest({
        node,
        global_config: {
          mdm: { enabled_and_configured: true },
          features: { enable_software_inventory: true },
        },
      });

      await user.click(screen.getByRole("button", { name: "Turn on MDM" }));
    });

    it("shows a banner when MDM is configured and the device doesn't have MDM info", async () => {
      const node = createMockNode() as INodeDevice;
      node.mdm.enrollment_status = null;
      node.platform = "darwin";
      node.dep_assigned_to_mdmlab = false;

      const user = await setupTest({
        node,
        global_config: {
          mdm: { enabled_and_configured: true },
          features: { enable_software_inventory: true },
        },
      });

      await user.click(screen.getByRole("button", { name: "Turn on MDM" }));
    });

    it("doesn't  show a banner when MDM is not configured", async () => {
      const node = createMockNode() as INodeDevice;
      node.mdm.enrollment_status = null;
      node.platform = "darwin";
      node.dep_assigned_to_mdmlab = false;

      await setupTest({
        node,
        global_config: {
          mdm: { enabled_and_configured: false },
          features: { enable_software_inventory: true },
        },
      });

      const btn = screen.queryByRole("button", { name: "Turn on MDM" });
      expect(btn).toBeNull();
    });

    it("doesn't  show a banner when the node already has MDM enabled", async () => {
      const node = createMockNode() as INodeDevice;
      node.mdm.enrollment_status = "On (manual)";
      node.platform = "darwin";
      node.dep_assigned_to_mdmlab = false;

      await setupTest({
        node,
        global_config: {
          mdm: { enabled_and_configured: true },
          features: { enable_software_inventory: true },
        },
      });

      const btn = screen.queryByRole("button", { name: "Turn on MDM" });
      expect(btn).toBeNull();
    });
  });
});
