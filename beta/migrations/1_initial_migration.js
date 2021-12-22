const Migrations = artifacts.require("Migrations");
const PhoenixClient = artifacts.require("phoenixClient");

module.exports = function (deployer) {
  deployer.deploy(Migrations);
  deployer.deploy(PhoenixClient);
};
