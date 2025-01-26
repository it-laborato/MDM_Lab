import React, { useState, useEffect, useRef } from "react";
import { useQuery } from "react-query";

import scriptsAPI, { IScriptResultResponse } from "services/entities/scripts";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";
import Textarea from "components/Textarea";
import DataError from "components/DataError/DataError";
import Spinner from "components/Spinner/Spinner";
import ModalFooter from "components/ModalFooter";

const baseClass = "run-script-details-modal";

interface IScriptContentProps {
  content: string;
}

const ScriptContent = ({ content }: IScriptContentProps) => {
  return (
    <div className={`${baseClass}__script-content`}>
      <span>Script content:</span>
      <Textarea className={`${baseClass}__script-content-textarea`}>
        {content}
      </Textarea>
    </div>
  );
};

const StatusMessageRunning = () => (
  <div className={`${baseClass}__status-message`}>
    <p>
      <Icon name="pending-outline" />
      Script is running or will run when the node comes online.
    </p>
  </div>
);

const StatusMessageSuccess = () => (
  <div className={`${baseClass}__status-message`}>
    <p>
      <Icon name="success-outline" />
      Exit code: 0 (Script ran successfully.)
    </p>
  </div>
);

const StatusMessageFailed = ({ exitCode }: { exitCode: number }) => (
  <div className={`${baseClass}__status-message`}>
    <p>
      <Icon name="error-outline" />
      Exit code: {exitCode} (Script failed.)
    </p>{" "}
  </div>
);

const StatusMessageError = ({ message }: { message: React.ReactNode }) => (
  <div className={`${baseClass}__status-message`}>
    <p>
      <Icon name="error-outline" />
      Error: {message}
    </p>
  </div>
);

interface IStatusMessageProps {
  nodeTimeout: boolean;
  exitCode: number | null;
  message: string;
}

const StatusMessage = ({
  nodeTimeout,
  exitCode,
  message,
}: IStatusMessageProps) => {
  switch (exitCode) {
    case null:
      return !nodeTimeout ? (
        // Expected API message: "A script is already running on this node. Please wait about 1 minute to let it finish."
        <StatusMessageRunning />
      ) : (
        // Expected API message: "Mdmlab hasn’t heard from the node in over 1 minute. Mdmlab doesn’t know if the script ran because the node went offline."
        <StatusMessageError message={message} />
      );
    case -2:
      // Expected API message: "Scripts are disabled for this node. To run scripts, deploy the mdmlabd agent with scripts enabled."
      return <StatusMessageError message={message} />;
    case -1: {
      // message should look like: "Timeout. Mdmlab stopped the script after 600 seconds to protect node performance.";
      const timeOutValue = message.match(/(\d+\s(?:seconds))/);

      // should always be there, but handle cleanly if not
      const varText = timeOutValue ? (
        <>
          after{" "}
          <TooltipWrapper tipContent="Timeout can be configured by updating agent options.">
            {timeOutValue[0]}
          </TooltipWrapper>{" "}
        </>
      ) : null;

      const modMessage = (
        <>
          Timeout. Mdmlab stopped the script {varText}to protect node
          performance.
        </>
      );
      return <StatusMessageError message={modMessage} />;
    }
    case 0:
      // Expected API message: ""
      return <StatusMessageSuccess />;
    default:
      // Expected API message: ""
      return <StatusMessageFailed exitCode={exitCode} />;
  }
};

interface IScriptOutputProps {
  output: string;
  nodename: string;
}

const ScriptOutput = ({ output, nodename }: IScriptOutputProps) => {
  return (
    <div className={`${baseClass}__script-output`}>
      <p>
        The{" "}
        <TooltipWrapper
          tipContent="Mdmlab records the last 10,000 characters to prevent downtime."
          tooltipClass={`${baseClass}__output-tooltip`}
          isDelayed
        >
          output recorded
        </TooltipWrapper>{" "}
        when <b>{nodename}</b> ran the script above:
      </p>
      <Textarea className={`${baseClass}__output-textarea`}>{output}</Textarea>
    </div>
  );
};

interface IScriptResultProps {
  nodename: string;
  output: string;
}

const ScriptResult = ({ nodename, output }: IScriptResultProps) => {
  return (
    <div className={`${baseClass}__script-result`}>
      <ScriptOutput output={output} nodename={nodename} />
    </div>
  );
};

interface IRunScriptDetailsModalProps {
  scriptExecutionId: string;
  onCancel: () => void;
  isHidden?: boolean;
}

const RunScriptDetailsModal = ({
  scriptExecutionId,
  onCancel,
  isHidden = false,
}: IRunScriptDetailsModalProps) => {
  // For scrollable modal
  const [isTopScrolling, setIsTopScrolling] = useState(false);
  const topDivRef = useRef<HTMLDivElement>(null);
  const checkScroll = () => {
    if (topDivRef.current) {
      const isScrolling =
        topDivRef.current.scrollHeight > topDivRef.current.clientHeight;
      setIsTopScrolling(isScrolling);
    }
  };

  const { data, isLoading, isError } = useQuery<IScriptResultResponse>(
    ["runScriptDetailsModal", scriptExecutionId],
    () => {
      return scriptsAPI.getScriptResult(scriptExecutionId);
    },
    { refetchOnWindowFocus: false, enabled: !!scriptExecutionId }
  );

  // For scrollable modal
  useEffect(() => {
    checkScroll();
    window.addEventListener("resize", checkScroll);
    return () => window.removeEventListener("resize", checkScroll);
  }, [data]); // Re-run when data changes

  const renderContent = () => {
    let content = <></>;

    if (isLoading) {
      content = <Spinner />;
    } else if (isError) {
      content = <DataError description="Close this modal and try again." />;
    } else if (data) {
      const nodeTimedOut =
        data.exit_code === null && data.node_timeout === true;
      const scriptsDisabledForNode = data.exit_code === -2;
      const scriptStillRunning =
        data.exit_code === null && data.node_timeout === false;
      const showOutputText =
        !nodeTimedOut && !scriptsDisabledForNode && !scriptStillRunning;

      content = (
        <>
          <StatusMessage
            nodeTimeout={data.node_timeout}
            exitCode={data.exit_code}
            message={data.output}
          />
          <ScriptContent content={data.script_contents} />
          {showOutputText && (
            <ScriptResult nodename={data.nodename} output={data.output} />
          )}
        </>
      );
    }

    return (
      <div
        className={`${baseClass}__modal-content modal-scrollable-content`}
        ref={topDivRef}
      >
        {content}
      </div>
    );
  };

  const renderFooter = () => (
    <ModalFooter
      isTopScrolling={isTopScrolling}
      primaryButtons={
        <Button onClick={onCancel} variant="brand">
          Done
        </Button>
      }
    />
  );
  return (
    <Modal
      title="Script details"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
      isHidden={isHidden}
    >
      <>
        {renderContent()}
        {renderFooter()}
      </>
    </Modal>
  );
};

export default RunScriptDetailsModal;
