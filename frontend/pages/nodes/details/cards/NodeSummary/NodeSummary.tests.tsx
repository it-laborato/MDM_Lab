import React from "react";
import { noop } from "lodash";
import { screen, fireEvent } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockUser from "__mocks__/userMock";
import { createMockNodeSummary } from "__mocks__/nodeMock";

import NodeSummary from "./NodeSummary";

describe("Node Summary section", () => {
  describe("Issues data", () => {
    it("omit issues header if no issues", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });
      const summaryData = createMockNodeSummary({});

      render(
        <NodeSummary
          summaryData={summaryData}
          showRefetchSpinner={false}
          onRefetchNode={noop}
          renderActionDropdown={() => null}
        />
      );

      expect(screen.queryByText("Issues")).not.toBeInTheDocument();
    });
  });
  describe("Agent data", () => {
    it("with all info present, render Agent header with orbit_version and tooltip with all 3 data points", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });
      const summaryData = createMockNodeSummary();
      const orbitVersion = summaryData.orbit_version as string;
      const osqueryVersion = summaryData.osquery_version as string;
      const mdmlabdVersion = summaryData.mdmlab_desktop_version as string;

      render(
        <NodeSummary
          summaryData={summaryData}
          showRefetchSpinner={false}
          onRefetchNode={noop}
          renderActionDropdown={() => null}
        />
      );

      expect(screen.getByText("Agent")).toBeInTheDocument();

      await fireEvent.mouseEnter(
        screen.getByText(new RegExp(orbitVersion, "i"))
      );

      expect(
        screen.getByText(new RegExp(osqueryVersion, "i"))
      ).toBeInTheDocument();
      expect(
        screen.getByText(new RegExp(mdmlabdVersion, "i"))
      ).toBeInTheDocument();
    });

    it("omit mdmlab desktop from tooltip if no mdmlab desktop version", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });
      const summaryData = createMockNodeSummary({
        mdmlab_desktop_version: null,
      });
      const orbitVersion = summaryData.orbit_version as string;
      const osqueryVersion = summaryData.osquery_version as string;

      render(
        <NodeSummary
          summaryData={summaryData}
          showRefetchSpinner={false}
          onRefetchNode={noop}
          renderActionDropdown={() => null}
        />
      );

      expect(screen.getByText("Agent")).toBeInTheDocument();

      await fireEvent.mouseEnter(
        screen.getByText(new RegExp(orbitVersion, "i"))
      );

      expect(
        screen.getByText(new RegExp(osqueryVersion, "i"))
      ).toBeInTheDocument();
      expect(screen.queryByText(/Mdmlab desktop:/i)).not.toBeInTheDocument();
    });

    it("for Chromebooks, render Agent header with osquery_version that is the mdmlabd chrome version and no tooltip", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });
      const summaryData = createMockNodeSummary({
        platform: "chrome",
        osquery_version: "mdmlabd-chrome 1.2.0",
      });

      const mdmlabdChromeVersion = summaryData.osquery_version as string;

      const { user } = render(
        <NodeSummary
          summaryData={summaryData}
          showRefetchSpinner={false}
          onRefetchNode={noop}
          renderActionDropdown={() => null}
        />
      );

      expect(screen.getByText("Agent")).toBeInTheDocument();
      await user.hover(screen.getByText(new RegExp(mdmlabdChromeVersion, "i")));
      expect(screen.queryByText("Osquery")).not.toBeInTheDocument();
    });
  });
  describe("iOS and iPadOS data", () => {
    it("for iOS, renders Team, Disk space, and Operating system data only", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const summaryData = createMockNodeSummary({
        team_id: 2,
        team_name: "Mobile",
        platform: "ios",
        os_version: "iOS 14.7.1",
      });

      const teamName = summaryData.team_name as string;
      const diskSpaceAvailable = summaryData.gigs_disk_space_available as string;
      const osVersion = summaryData.os_version as string;

      render(
        <NodeSummary
          summaryData={summaryData}
          showRefetchSpinner={false}
          onRefetchNode={noop}
          renderActionDropdown={() => null}
          isPremiumTier
        />
      );

      expect(screen.getByText("Team").nextElementSibling).toHaveTextContent(
        teamName
      );
      expect(
        screen.getByText("Disk space").nextElementSibling
      ).toHaveTextContent(`${diskSpaceAvailable} GB available`);
      expect(
        screen.getByText("Operating system").nextElementSibling
      ).toHaveTextContent(osVersion);
      expect(screen.queryByText("Refetch")).toBeInTheDocument();

      expect(screen.queryByText("Status")).not.toBeInTheDocument();
      expect(screen.queryByText("Memory")).not.toBeInTheDocument();
      expect(screen.queryByText("Processor type")).not.toBeInTheDocument();
      expect(screen.queryByText("Agent")).not.toBeInTheDocument();
      expect(screen.queryByText("Osquery")).not.toBeInTheDocument();
    });
    it("for iPadOS, renders Team, Disk space, and Operating system data only", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const summaryData = createMockNodeSummary({
        team_id: 2,
        team_name: "Mobile",
        platform: "ipados",
        os_version: "iPadOS 16.7.8",
      });

      const teamName = summaryData.team_name as string;
      const diskSpaceAvailable = summaryData.gigs_disk_space_available as string;
      const osVersion = summaryData.os_version as string;

      render(
        <NodeSummary
          summaryData={summaryData}
          showRefetchSpinner={false}
          onRefetchNode={noop}
          renderActionDropdown={() => null}
          isPremiumTier
        />
      );

      expect(screen.getByText("Team").nextElementSibling).toHaveTextContent(
        teamName
      );
      expect(
        screen.getByText("Disk space").nextElementSibling
      ).toHaveTextContent(`${diskSpaceAvailable} GB available`);
      expect(
        screen.getByText("Operating system").nextElementSibling
      ).toHaveTextContent(osVersion);
      expect(screen.queryByText("Refetch")).toBeInTheDocument();

      expect(screen.queryByText("Status")).not.toBeInTheDocument();
      expect(screen.queryByText("Memory")).not.toBeInTheDocument();
      expect(screen.queryByText("Processor type")).not.toBeInTheDocument();
      expect(screen.queryByText("Agent")).not.toBeInTheDocument();
      expect(screen.queryByText("Osquery")).not.toBeInTheDocument();
    });
  });
  describe("Maintenance window data", () => {
    it("renders maintenance window data with timezone", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });

      const summaryData = createMockNodeSummary({
        maintenance_window: {
          starts_at: "3025-06-24T20:48:14-03:00",
          timezone: "America/Argentina/Buenos_Aires",
        },
      });
      const prettyStartTime = /Jun 24 at 8:48 PM/;

      render(
        <NodeSummary
          summaryData={summaryData}
          showRefetchSpinner={false}
          onRefetchNode={noop}
          renderActionDropdown={() => null}
          isPremiumTier
        />
      );

      expect(screen.getByText("Scheduled maintenance")).toBeInTheDocument();
      expect(screen.getByText(prettyStartTime)).toBeInTheDocument();
    });
  });
});
