import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockRouter } from "test/test-utils";

import { ISoftwareResponse } from "interfaces/software";

import Software from "./Software";

describe("Dashboard software card", () => {
  const vulnSwInfo = {
    name: "ms-toolsai.jupyter",
    version: "2.3.4",
    nodesCount: 432,
  };
  const noVulnSwInfo = {
    name: "common.extension_2",
    version: "5.6.7",
    nodesCount: 543,
  };

  it("renders all software normally when present", () => {
    const allSwResponse: ISoftwareResponse = {
      counts_updated_at: "2024-08-21T21:01:31Z",
      software: [
        {
          id: 784,
          name: vulnSwInfo.name,
          version: vulnSwInfo.version,
          source: "vscode_extensions",
          browser: "",
          vendor: "Microsoft",
          generated_cpe:
            "cpe:2.3:a:microsoft:jupyter:2023.10.10:*:*:*:*:visual_studio_code:*:*",
          vulnerabilities: [
            {
              cve: "CVE-2023-36018",
              details_link: "https://nvd.nist.gov/vuln/detail/CVE-2023-36018",
              created_at: "2024-07-12T04:00:56Z",
              cvss_score: 9.8,
              epss_probability: 0.00162,
              cisa_known_exploit: false,
              cve_published: "2023-11-14T18:15:00Z",
              cve_description:
                "Visual Studio Code Jupyter Extension Spoofing Vulnerability",
              resolved_in_version: "2023.10.1100000000",
            },
          ],
          nodes_count: vulnSwInfo.nodesCount,
        },
        {
          id: 758,
          name: noVulnSwInfo.name,
          version: noVulnSwInfo.version,
          source: "vscode_extensions",
          browser: "",
          generated_cpe: "",
          vulnerabilities: null,
          nodes_count: noVulnSwInfo.nodesCount,
        },
      ],
    };
    render(
      <Software
        errorSoftware={null}
        isSoftwareFetching={false}
        isSoftwareEnabled
        navTabIndex={0}
        onTabChange={jest.fn()}
        onQueryChange={jest.fn()}
        software={allSwResponse}
        teamId={-1}
        router={createMockRouter()}
      />
    );

    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("Version")).toBeInTheDocument();
    expect(screen.getByText("Nodes")).toBeInTheDocument();

    Object.keys(noVulnSwInfo).forEach((key) => {
      expect(
        screen.getByText(noVulnSwInfo[key as keyof typeof noVulnSwInfo])
      ).toBeInTheDocument();
    });
  });
  it("renders vulnerable software normally when present", () => {
    const vulnSwResponse: ISoftwareResponse = {
      counts_updated_at: "2024-08-21T21:01:31Z",
      software: [
        {
          id: 784,
          name: vulnSwInfo.name,
          version: vulnSwInfo.version,
          source: "vscode_extensions",
          browser: "",
          vendor: "Microsoft",
          generated_cpe:
            "cpe:2.3:a:microsoft:jupyter:2023.10.10:*:*:*:*:visual_studio_code:*:*",
          vulnerabilities: [
            {
              cve: "CVE-2023-36018",
              details_link: "https://nvd.nist.gov/vuln/detail/CVE-2023-36018",
              created_at: "2024-07-12T04:00:56Z",
              cvss_score: 9.8,
              epss_probability: 0.00162,
              cisa_known_exploit: false,
              cve_published: "2023-11-14T18:15:00Z",
              cve_description:
                "Visual Studio Code Jupyter Extension Spoofing Vulnerability",
              resolved_in_version: "2023.10.1100000000",
            },
          ],
          nodes_count: vulnSwInfo.nodesCount,
        },
      ],
    };
    render(
      <Software
        errorSoftware={null}
        isSoftwareFetching={false}
        isSoftwareEnabled
        navTabIndex={1}
        onTabChange={jest.fn()}
        onQueryChange={jest.fn()}
        software={vulnSwResponse}
        teamId={-1}
        router={createMockRouter()}
      />
    );

    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("Version")).toBeInTheDocument();
    expect(screen.getByText("Nodes")).toBeInTheDocument();

    Object.keys(vulnSwInfo).forEach((key) => {
      expect(
        screen.getByText(vulnSwInfo[key as keyof typeof vulnSwInfo])
      ).toBeInTheDocument();
    });
  });
});
