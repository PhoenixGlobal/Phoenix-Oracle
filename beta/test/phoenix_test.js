const testApi = require('./support/helpers.js')

contract('phoenixClient', () => {
    let phoenix = artifacts.require("../contracts/phoenixClient.sol");
    let GetterSetter = artifacts.require("../contracts/GetterSetter.sol");
    let oc;
    let fID = "0x12345678";
    let to = "0x80E29AcB842498fE6591F020bd82766DCe619D43";
    beforeEach(async () => {
        oc = await phoenix.new({from : oracle});
        // let hexOfval = web3.utils.toHex("hello world");
        // let data32 = web3.utils.hexToBytes(hexOfval,32)
        // console.log("ffffffffffffff")
        // console.log(data32)
        // console.log("ffffffffffffff")
    });

    it("has a limited public interface", () => {
        testApi.checkPublicABI(phoenix, [
            "transferOwnership",
            "requestData",
            "fulfillData",
        ]);
    });

    describe("#transferOwnership", () => {
        context("when called by the owner", () => {
            beforeEach( async () => {
                await oc.transferOwnership(stranger, {from: oracle});
            });

            it("can change the owner", async () => {
                let owner = await oc.owner.call();
                assert.isTrue(web3.utils.isAddress(owner));
                assert.equal(stranger, owner);
            });
        });

        context("when called by a non-owner", () => {
            it("cannot change the owner", async () => {
                await testApi.assertActionThrows(async () => {
                    await oc.transferOwnership(stranger, {from: stranger});
                });
            });
        });
    });

    describe("#requestData", () => {
        it("logs an event", async () => {
            let tx = await oc.requestData(to, fID);
            assert.equal(1, tx.receipt.logs.length)
            console.log("1111111111111111111111111111")
            console.log(tx.receipt.logs[0].args)


            let log = tx.receipt.logs[0]
            assert.equal(to, testApi.hexToAddress(log.args[1]))
        });

        it("increments the nonce", async () => {
            let tx1 = await oc.requestData(to, fID);
            let nonce1 = web3.utils.toDecimal(tx1.receipt.logs[0].args[0]);
            let tx2 = await oc.requestData(to, fID);
            let nonce2 = web3.utils.toDecimal(tx2.receipt.logs[0].args[0]);

            assert.notEqual(nonce1, nonce2);
        });
    });

    describe("#fulfillData", () => {
        let mock, nonce;

        beforeEach(async () => {
            mock = await GetterSetter.new();
            console.log("22222222222222222222")
            console.log(mock.address)


        });

        // context("when the called by a non-owner", () => {
        //     it("raises an error", async () => {
        //         await testApi.assertActionThrows(async () => {
        //             await oc.fulfillData(nonce, "Hello World!", {from: stranger});
        //         });
        //     });
        // });
        //
        // context("when called by an owner", () => {
        //     it("raises an error if the request ID does not exist", async () => {
        //         await testApi.assertActionThrows(async () => {
        //             await oc.fulfillData(nonce + 1, "Hello World!", {from: oracle});
        //         });
        //     });
        //
            it("sets the value on the requested contract", async () => {
                let funcId =  testApi.functionID("setValue(bytes32)")
                console.log(funcId)
                let req = await oc.requestData(mock.address, "0x58825b10");
                nonce = web3.utils.toDecimal(req.receipt.logs[0].args[0]);
                let hexOfVal = web3.utils.stringToHex("hello apex");
                console.log(hexOfVal);
                await oc.fulfillData(nonce, hexOfVal, {from: oracle});
                let current = await mock.value.call();
                console.log(current)
                assert.equal("hello apex", web3.utils.toUtf8(current));
            });

        //     it("does not allow a request to be fulfilled twice", async () => {
        //         await oc.fulfillData(nonce, "First message!", {from: oracle});
        //         await testApi.assertActionThrows(async () => {
        //             await oc.fulfillData(nonce, "Second message!!", {from: oracle});
        //         });
        //     });
        // });
    });
});