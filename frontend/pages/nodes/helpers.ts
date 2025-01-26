// eslint-disable-next-line import/prefer-default-export
const getNodeStatusTooltipText = (status: string): string => {
  if (status === "online") {
    return "Online nodes will respond to a live query.";
  }
  return "Offline nodes won’t respond to a live query because they may be shut down, asleep, or not connected to the internet.";
};

export default getNodeStatusTooltipText;
