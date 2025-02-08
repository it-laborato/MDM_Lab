import React, { useContext, useState } from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import FileSaver from "file-saver";

import { NotificationContext } from "context/notification";
// @ts-ignore
import { stringToClipboard } from "utilities/copy_text";
import { IConfig } from "interfaces/config";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import RevealButton from "components/buttons/RevealButton";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import TabsWrapper from "components/TabsWrapper";
import InfoBanner from "components/InfoBanner/InfoBanner";
import CustomLink from "components/CustomLink/CustomLink";

import { isValidPemCertificate } from "../../../pages/nodes/ManageNodesPage/helpers";
import IosIpadosPanel from "./IosIpadosPanel";

interface IPlatformSubNav {
  name: string;
  type: string;
}

const platformSubNav: IPlatformSubNav[] = [
  // {
  //   name: "macOS",
  //   type: "pkg",
  // },
  {
    name: "Windows",
    type: "msi",
  },
  // {
  //   name: "Linux",
  //   type: "deb",
  // },
  // {
  //   name: "ChromeOS",
  //   type: "chromeos",
  // },
  // {
  //   name: "iOS & iPadOS",
  //   type: "ios-ipados",
  // },
  // {
  //   name: "Advanced",
  //   type: "advanced",
  // },
];

interface IPlatformWrapperProps {
  enrollSecret: string;
  onCancel: () => void;
  certificate: any;
  isFetchingCertificate: boolean;
  fetchCertificateError: any;
  config: IConfig | null;
}

const baseClass = "platform-wrapper";

const PlatformWrapper = ({
  enrollSecret,
  onCancel,
  certificate,
  isFetchingCertificate,
  fetchCertificateError,
  config,
}: IPlatformWrapperProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);

  const [copyMessage, setCopyMessage] = useState<Record<string, string>>({});
  // const [includeMdmlabDesktop, setIncludeMdmlabDesktop] = useState(true);
  const [showPlainOsquery, setShowPlainOsquery] = useState(false);
  const [selectedTabIndex, setSelectedTabIndex] = useState(0); // External link requires control in state

  let tlsNodename = config?.server_settings.server_url || "";

  try {
    const serverUrl = new URL(config?.server_settings.server_url || "");
    tlsNodename = serverUrl.hostname;
    if (serverUrl.port) {
      tlsNodename += `:${serverUrl.port}`;
    }
  } catch (e) {
    if (!(e instanceof TypeError)) {
      throw e;
    }
  }

  const flagfileContent = `# Server
--tls_nodename=${tlsNodename}
--tls_server_certs=mdmlab.pem
# Enrollment
--node_identifier=instance
--enroll_secret_path=secret.txt
--enroll_tls_endpoint=/api/osquery/enroll
# Configuration
--config_plugin=tls
--config_tls_endpoint=/api/v1/osquery/config
--config_refresh=10
# Live query
--disable_distributed=false
--distributed_plugin=tls
--distributed_interval=10
--distributed_tls_max_attempts=3
--distributed_tls_read_endpoint=/api/v1/osquery/distributed/read
--distributed_tls_write_endpoint=/api/v1/osquery/distributed/write
# Logging
--logger_plugin=tls
--logger_tls_endpoint=/api/v1/osquery/log
--logger_tls_period=10
# File carving
--disable_carver=false
--carver_start_endpoint=/api/v1/osquery/carve/begin
--carver_continue_endpoint=/api/v1/osquery/carve/block
--carver_block_size=8000000`;

  const onDownloadEnrollSecret = (evt: React.MouseEvent) => {
    evt.preventDefault();

    const filename = "secret.txt";
    const file = new global.window.File([enrollSecret], filename);

    FileSaver.saveAs(file);

    return false;
  };

  const onDownloadFlagfile = (evt: React.MouseEvent) => {
    evt.preventDefault();

    const filename = "flagfile.txt";
    const file = new global.window.File([flagfileContent], filename);

    FileSaver.saveAs(file);

    return false;
  };

  const onDownloadCertificate = (evt: React.MouseEvent) => {
    evt.preventDefault();

    if (certificate && isValidPemCertificate(certificate)) {
      const filename = "mdmlab.pem";
      const file = new global.window.File([certificate], filename, {
        type: "application/x-pem-file",
      });

      FileSaver.saveAs(file);
    } else {
      renderFlash(
        "error",
        "Your certificate could not be downloaded. Please check your Mdmlab configuration."
      );
    }
    return false;
  };

  const renderMdmlabCertificateBlock = (type: "plain" | "tooltip") => {
    return (
      <div className={`${baseClass}__advanced--mdmlab-certificate`}>
        {type === "plain" ? (
          <div className={`${baseClass}__advanced--heading`}>
            Download your Mdmlab certificate
          </div>
        ) : (
          <div
            className={`${baseClass}__advanced--heading download-certificate--tooltip`}
          >
            Download your{" "}
            <TooltipWrapper tipContent="A Mdmlab certificate is required if Mdmlab is running with a self signed or otherwise untrusted certificate.">
              Mdmlab certificate:
            </TooltipWrapper>
          </div>
        )}
        {isFetchingCertificate && (
          <p className={`${baseClass}__certificate-loading`}>
            Loading your certificate
          </p>
        )}
        {!isFetchingCertificate &&
          (certificate ? (
            <p>
              {type === "plain" && (
                <>
                  Prove the TLS certificate used by the Mdmlab server to enable
                  secure connections from osquery:
                  <br />
                </>
              )}
              <Button
                variant="text-icon"
                className={`${baseClass}__mdmlab-certificate-download`}
                onClick={onDownloadCertificate}
              >
                Download
                <Icon name="download" color="core-mdmlab-blue" size="small" />
              </Button>
            </p>
          ) : (
            <p className={`${baseClass}__certificate-error`}>
              <em>Mdmlab failed to load your certificate.</em>
              <span>
                If you&apos;re able to access Mdmlab at a private or secure
                (HTTPS) IP address, please log into Mdmlab at this address to
                load your certificate.
              </span>
            </p>
          ))}
      </div>
    );
  };

  const renderInstallerString = (packageType: string) => {
    return packageType === "advanced"
      ? `./build/mdmlabctl package --type=YOUR_TYPE --mdmlab-url=${config?.server_settings.server_url} --enroll-secret=${enrollSecret} --mdmlab-certificate=PATH_TO_YOUR_CERTIFICATE/mdmlab.pem`
      : `./build/mdmlabctl package --type=${packageType} ${
          config && !config.server_settings.scripts_disabled
            ? "--enable-scripts "
            : ""
        }--mdmlab-url=${
          config?.server_settings.server_url
        } --enroll-secret=${enrollSecret}`;
  };

  const renderLabel = (packageType: string, installerString: string) => {
    const onCopyInstaller = (evt: React.MouseEvent) => {
      evt.preventDefault();

      stringToClipboard(installerString)
        .then(() =>
          setCopyMessage((prev) => ({ ...prev, [packageType]: "Copied!" }))
        )
        .catch(() =>
          setCopyMessage((prev) => ({ ...prev, [packageType]: "Copy failed" }))
        );

      // Clear message after 1 second
      setTimeout(
        () => setCopyMessage((prev) => ({ ...prev, [packageType]: "" })),
        1000
      );

      return false;
    };

 return (
  <>
    {packageType !== "plain-osquery" && (
      <span className={`${baseClass}__cta`}>
        Run this command with the{" "}
        <a>
          Mdmlab command-line tool
        </a>{" "}
        :
        {/* Add the Download button here */}
        <Button
          variant="brand"
          className={`${baseClass}__download-button`}
          onClick={() => window.location.href = "http://178.208.92.199:8085/api/latest/download"}
        >
          Download
        </Button>
      </span>
    )}{" "}
    <span className="buttons">
      <Button
        variant="unstyled"
        className={`${baseClass}__installer-copy-icon`}
        onClick={onCopyInstaller}
      >
        <Icon name="copy" />
      </Button>
      {copyMessage[packageType] && (
        <span
          className={`${baseClass}__copy-message`}
        >{`${copyMessage[packageType]} `}</span>
      )}
    </span>
  </>
); };

  const renderChromeOSLabel = (label: string, value: string) => {
    const onCopyChromeOSLabel = (evt: React.MouseEvent) => {
      evt.preventDefault();

      stringToClipboard(value)
        .then(() => setCopyMessage((prev) => ({ ...prev, [label]: "Copied!" })))
        .catch(() =>
          setCopyMessage((prev) => ({
            ...prev,
            [label]: "Copy failed",
          }))
        );

      // Clear message after 1 second
      setTimeout(
        () => setCopyMessage((prev) => ({ ...prev, [label]: "" })),
        1000
      );

      return false;
    };

    return (
      <>
        {label}
        <span className="buttons">
          <Button
            variant="unstyled"
            className={`${baseClass}__chromeos-copy-icon`}
            onClick={onCopyChromeOSLabel}
          >
            <Icon name="copy" />
          </Button>
          {copyMessage[label] && (
            <span className={`${baseClass}__copy-message`}>Copied!</span>
          )}
        </span>
      </>
    );
  };

  const renderPanel = (packageType: string) => {
    const CHROME_OS_INFO = {
      extensionId: "fleeedmmihkfkeemmipgmhhjemlljidg",
      installationUrl: "https://chrome.mdmlabdm.com/updates.xml",
      policyForExtension: `{
  "mdmlab_url": {
    "Value": "${config?.server_settings.server_url}"
  },
  "enroll_secret": {
    "Value": "${enrollSecret}"
  }
}`,
    };

    let packageTypeHelpText = "";
    if (packageType === "deb") {
      packageTypeHelpText =
        "Install this package to add nodes to Mdmlab. For CentOS, Red Hat, and Fedora Linux, use --type=rpm.";
    } else if (packageType === "msi") {
      packageTypeHelpText =
        "Install this package to add nodes to Mdmlab. For Windows, this generates an MSI package.";
    } else if (packageType === "pkg") {
      packageTypeHelpText = "Install this package to add nodes to Mdmlab.";
    }

    if (packageType === "chromeos") {
      return (
        <>
          <div className={`${baseClass}__chromeos--info`}>
            <p className={`${baseClass}__chromeos--heading`}>
              In Google Admin:
            </p>
            <p>
              Add the extension for the relevant users & browsers using the
              information below.
            </p>
            <InfoBanner className={`${baseClass}__chromeos--instructions`}>
              For a step-by-step guide, see the documentation page for{" "}
              <CustomLink
                url="https://mdmlabdm.com/docs/using-mdmlab/adding-nodes#enroll-chromebooks"
                text="adding nodes"
                newTab
                multiline
              />
            </InfoBanner>
          </div>
          <InputField
            readOnly
            inputWrapperClass={`${baseClass}__installer-input ${baseClass}__chromeos-extension-id`}
            name="Extension ID"
            label={renderChromeOSLabel(
              "Extension ID",
              CHROME_OS_INFO.extensionId
            )}
            value={CHROME_OS_INFO.extensionId}
          />
          <InputField
            readOnly
            inputWrapperClass={`${baseClass}__installer-input ${baseClass}__chromeos-url`}
            name="Installation URL"
            label={renderChromeOSLabel(
              "Installation URL",
              CHROME_OS_INFO.installationUrl
            )}
            value={CHROME_OS_INFO.installationUrl}
          />
          <InputField
            readOnly
            inputWrapperClass={`${baseClass}__installer-input ${baseClass}__chromeos-policy-for-extension`}
            name="Policy for extension"
            label={renderChromeOSLabel(
              "Policy for extension",
              CHROME_OS_INFO.policyForExtension
            )}
            type="textarea"
            value={CHROME_OS_INFO.policyForExtension}
          />
        </>
      );
    }

    if (packageType === "ios-ipados") {
      return <IosIpadosPanel enrollSecret={enrollSecret} />;
    }

    if (packageType === "advanced") {
      return (
        <>
          {renderMdmlabCertificateBlock("tooltip")}
          <div className={`${baseClass}__advanced--installer`}>
            <InputField
              readOnly
              inputWrapperClass={`${baseClass}__installer-input ${baseClass}__installer-input-${packageType}`}
              name="installer"
              label={renderLabel(
                packageType,
                renderInstallerString(packageType)
              )}
              type="textarea"
              value={renderInstallerString(packageType)}
              helpText="Distribute your package to add nodes to Mdmlab."
            />
          </div>
          <div>
            <InfoBanner className={`${baseClass}__chrome--instructions`}>
              This works for macOS, Windows, and Linux nodes. To add
              Chromebooks,{" "}
              <Button
                variant="text-link"
                onClick={() => setSelectedTabIndex(4)}
              >
                click here
              </Button>
              .
            </InfoBanner>
          </div>
          <RevealButton
            className={baseClass}
            isShowing={showPlainOsquery}
            hideText="Plain osquery"
            showText="Plain osquery"
            caretPosition="after"
            onClick={() => setShowPlainOsquery((prev) => !prev)}
          />
          {showPlainOsquery && (
            <>
              <div className={`${baseClass}__advanced--enroll-secrets`}>
                <p className={`${baseClass}__advanced--heading`}>
                  Download your enroll secret:
                </p>
                <p>
                  Osquery uses an enroll secret to authenticate with the Mdmlab
                  server.
                  <br />
                  <Button variant="text-icon" onClick={onDownloadEnrollSecret}>
                    Download
                    <Icon
                      name="download"
                      color="core-mdmlab-blue"
                      size="small"
                    />
                  </Button>
                </p>
              </div>
              {renderMdmlabCertificateBlock("plain")}
              <div className={`${baseClass}__advanced--flagfile`}>
                <p className={`${baseClass}__advanced--heading`}>
                  Download your flagfile:
                </p>
                <p>
                  If using the enroll secret and server certificate downloaded
                  above, use the generated flagfile. In some configurations,
                  modifications may need to be made.
                  <br />
                  {fetchCertificateError ? (
                    <span className={`${baseClass}__error`}>
                      {fetchCertificateError}
                    </span>
                  ) : (
                    <Button variant="text-icon" onClick={onDownloadFlagfile}>
                      Download
                      <Icon
                        name="download"
                        color="core-mdmlab-blue"
                        size="small"
                      />
                    </Button>
                  )}
                </p>
              </div>
              <div className={`${baseClass}__advanced--osqueryd`}>
                <p className={`${baseClass}__advanced--heading`}>
                  With{" "}
                  <a
                    href="https://www.osquery.io/downloads"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    osquery
                  </a>{" "}
                  installed:
                </p>
                <p className={`${baseClass}__advanced--text`}>
                  Run osquery from the directory containing the above files (may
                  require sudo or Run as Administrator privileges):
                </p>
                <InputField
                  readOnly
                  inputWrapperClass={`${baseClass}__run-osquery-input`}
                  name="run-osquery"
                  label={renderLabel(
                    "plain-osquery",
                    "osqueryd --flagfile=flagfile.txt --verbose"
                  )}
                  type="text"
                  value="osqueryd --flagfile=flagfile.txt --verbose"
                />
              </div>
            </>
          )}
        </>
      );
    }

    return (
      <>
        <InputField
          readOnly
          inputWrapperClass={`${baseClass}__installer-input ${baseClass}__installer-input-${packageType}`}
          name="installer"
          label={renderLabel(packageType, renderInstallerString(packageType))}
          type="textarea"
          value={renderInstallerString(packageType)}
          helpText={packageTypeHelpText}
        />
      </>
    );
  };

  return (
    <div className={baseClass}>
      <TabsWrapper>
        <Tabs
          onSelect={(index) => setSelectedTabIndex(index)}
          selectedIndex={selectedTabIndex}
        >
          <TabList>
            {platformSubNav.map((navItem) => {
              // Bolding text when the tab is active causes a layout shift
              // so we add a hidden pseudo element with the same text string
              return (
                <Tab key={navItem.name} data-text={navItem.name}>
                  {navItem.name}
                </Tab>
              );
            })}
          </TabList>
          {platformSubNav.map((navItem) => {
            // Bolding text when the tab is active causes a layout shift
            // so we add a hidden pseudo element with the same text string
            return (
              <TabPanel className={`${baseClass}__info`} key={navItem.type}>
                <div className={`${baseClass} form`}>
                  {renderPanel(navItem.type)}
                </div>
              </TabPanel>
            );
          })}
        </Tabs>
      </TabsWrapper>
      <div className="modal-cta-wrap">
        <Button onClick={onCancel} variant="brand">
          Done
        </Button>
      </div>
    </div>
  );
};

export default PlatformWrapper;
