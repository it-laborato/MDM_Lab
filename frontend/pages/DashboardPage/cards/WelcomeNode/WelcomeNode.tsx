import React, { useContext, useState } from "react";
import PATHS from "router/paths";
import { Link } from "react-router";
import { useQuery } from "react-query";
import { formatDistanceToNow } from "date-fns";

import { NotificationContext } from "context/notification";
import { INode, INodeResponse } from "interfaces/node";
import { INodePolicy } from "interfaces/policy";
import nodeAPI from "services/entities/nodes";

import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import Icon from "components/Icon/Icon";
import LaptopMac from "../../../../../assets/images/laptop-mac.png";
import SlackButton from "../../../../../assets/images/slack-button-get-help.png";

interface IWelcomeNodeCardProps {
  totalsNodesCount: number;
  toggleAddNodesModal: (showAddNodesModal: boolean) => void;
}

const baseClass = "welcome-node";
const HOST_ID = 1;
const POLICY_PASS = "pass";
const POLICY_FAIL = "fail";

const WelcomeNode = ({
  totalsNodesCount,
  toggleAddNodesModal,
}: IWelcomeNodeCardProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);
  const [refetchStartTime, setRefetchStartTime] = useState<number | null>(null);
  const [currentPolicyShown, setCurrentPolicyShown] = useState<INodePolicy>();
  const [showPolicyModal, setShowPolicyModal] = useState(false);
  const [isPoliciesEmpty, setIsPoliciesEmpty] = useState(false);
  const [showRefetchLoadingSpinner, setShowRefetchLoadingSpinner] = useState(
    false
  );

  const {
    isLoading: isLoadingNode,
    data: node,
    error: loadingNodeError,
    refetch: fullyReloadNode,
  } = useQuery<INodeResponse, Error, INode>(
    ["node"],
    () => nodeAPI.loadNodeDetails(HOST_ID),
    {
      retry: false,
      select: (data: INodeResponse) => data.node,
      onSuccess: (returnedNode) => {
        setShowRefetchLoadingSpinner(returnedNode.refetch_requested);

        const anyPassingOrFailingPolicy = returnedNode?.policies?.find(
          (p) => p.response === POLICY_PASS || p.response === POLICY_FAIL
        );
        setIsPoliciesEmpty(typeof anyPassingOrFailingPolicy === "undefined");

        if (returnedNode.refetch_requested) {
          // Code duplicated from NodeDetailsPage. See comments there.
          if (!refetchStartTime) {
            if (returnedNode.status === "online") {
              setRefetchStartTime(Date.now());
              setTimeout(() => {
                fullyReloadNode();
              }, 1000);
            } else {
              setShowRefetchLoadingSpinner(false);
            }
          } else {
            const totalElapsedTime = Date.now() - refetchStartTime;
            if (totalElapsedTime < 60000) {
              if (returnedNode.status === "online") {
                setTimeout(() => {
                  fullyReloadNode();
                }, 1000);
              } else {
                renderFlash(
                  "error",
                  `This node is offline. Please try refetching node vitals later.`
                );
                setShowRefetchLoadingSpinner(false);
              }
            } else {
              renderFlash(
                "error",
                `We're having trouble fetching fresh vitals for this node. Please try again later.`
              );
              setShowRefetchLoadingSpinner(false);
            }
          }
        }
      },
      onError: (error) => {
        console.error(error);
      },
    }
  );

  const onRefetchNode = async () => {
    if (node) {
      setShowRefetchLoadingSpinner(true);

      try {
        await nodeAPI.refetch(node).then(() => {
          setRefetchStartTime(Date.now());
          setTimeout(() => fullyReloadNode(), 1000);
        });
      } catch (error) {
        console.error(error);
        renderFlash("error", `Node "${node.display_name}" refetch error`);
        setShowRefetchLoadingSpinner(false);
      }
    }
  };

  const handlePolicyModal = (id: number) => {
    const policy = node?.policies.find((p) => p.id === id);

    if (policy) {
      setCurrentPolicyShown(policy);
      setShowPolicyModal(true);
    }
  };

  if (isLoadingNode) {
    return (
      <div className={baseClass}>
        <div className={`${baseClass}__loading`}>
          <Spinner />
        </div>
      </div>
    );
  }

  if (loadingNodeError) {
    return (
      <div className={baseClass}>
        <div className={`${baseClass}__empty-nodes`}>
          <p>Add your personal device to assess the security of your device.</p>
          <p>
            In Mdmlab, laptops, workstations, and servers are referred to as
            &quot;nodes.&quot;
          </p>
          <Button
            onClick={toggleAddNodesModal}
            className={`${baseClass}__add-node`}
            variant="brand"
          >
            <span>Add nodes</span>
          </Button>
        </div>
      </div>
    );
  }

  if (totalsNodesCount === 1 && node && node.status === "offline") {
    return (
      <div className={baseClass}>
        <div className={`${baseClass}__error`}>
          <p className="error-message">
            <Icon name="disable" color="status-error" />
            Your device is not communicating with Mdmlab.
          </p>
          <p>Join the #mdmlab Slack channel for help troubleshooting.</p>
          <a
            target="_blank"
            rel="noreferrer"
            href="https://osquery.slack.com/archives/C01DXJL16D8"
          >
            <img
              alt="Get help on Slack"
              className="button-slack"
              src={SlackButton}
            />
          </a>
        </div>
      </div>
    );
  }

  if (isPoliciesEmpty) {
    return (
      <div className={baseClass}>
        <div className={`${baseClass}__error`}>
          <p className="error-message">
            <Icon name="disable" color="status-error" />
            No policies apply to your device.
          </p>
          <p>Join the #mdmlab Slack channel for help troubleshooting.</p>
          <a
            target="_blank"
            rel="noreferrer"
            href="https://osquery.slack.com/archives/C01DXJL16D8"
          >
            <img
              alt="Get help on Slack"
              className="button-slack"
              src={SlackButton}
            />
          </a>
        </div>
      </div>
    );
  }

  if (totalsNodesCount === 1 && node && node.status === "online") {
    return (
      <div className={baseClass}>
        <div className={`${baseClass}__intro`}>
          <img alt="" src={LaptopMac} />
          <div className="info">
            <Link to={PATHS.HOST_DETAILS(node.id)} className="external-link">
              {node.display_name}
              <Icon name="arrow-internal-link" />
            </Link>
            <p>Your node is successfully connected to Mdmlab.</p>
          </div>
        </div>
        <div className={`${baseClass}__blurb`}>
          <p>
            Mdmlab already ran the following policies to assess the security of
            your device:{" "}
          </p>
        </div>
        <div className={`${baseClass}__policies`}>
          {node.policies?.slice(0, 3).map((p) => {
            if (p.response) {
              return (
                <Button
                  variant="unstyled"
                  onClick={() => handlePolicyModal(p.id)}
                >
                  <div className="policy-block">
                    <Icon
                      name={
                        p.response === POLICY_PASS ? "success" : "error-outline"
                      }
                    />
                    <span className="info">{p.name}</span>
                    <Icon
                      name="chevron-right"
                      color="core-mdmlab-blue"
                      className="policy-arrow"
                    />
                  </div>
                </Button>
              );
            }

            return null;
          })}
          {node.policies?.length > 3 && (
            <Link to={PATHS.HOST_POLICIES(node.id)} className="external-link">
              Go to Node details to see all policies
              <Icon name="arrow-internal-link" />
            </Link>
          )}
        </div>
        <div className={`${baseClass}__blurb`}>
          <p>Resolved a failing policy? Refetch your node vitals to verify.</p>
        </div>
        <div className={`${baseClass}__refetch`}>
          <Button
            variant="blue-green"
            className={`refetch-spinner ${
              showRefetchLoadingSpinner ? "spin" : ""
            }`}
            onClick={onRefetchNode}
            disabled={showRefetchLoadingSpinner}
          >
            <Icon name="refresh" color="core-mdmlab-white" /> Refetch
          </Button>
          <span>
            Last updated{" "}
            {formatDistanceToNow(new Date(node.detail_updated_at), {
              addSuffix: true,
            })}
          </span>
        </div>
        {showPolicyModal && (
          <Modal
            title={currentPolicyShown?.name || ""}
            onExit={() => setShowPolicyModal(false)}
            onEnter={() => setShowPolicyModal(false)}
            className={`${baseClass}__policy-modal`}
          >
            <>
              <p>{currentPolicyShown?.description}</p>
              {currentPolicyShown?.resolution && (
                <p>
                  <b>Resolve:</b> {currentPolicyShown.resolution}
                </p>
              )}
              <div className="modal-cta-wrap">
                <Button
                  variant="brand"
                  onClick={() => setShowPolicyModal(false)}
                >
                  Done
                </Button>
              </div>
            </>
          </Modal>
        )}
      </div>
    );
  }

  return <Spinner />;
};

export default WelcomeNode;
