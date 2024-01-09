import express, { Router, Request, Response, NextFunction } from 'express';
import { PrismaClient } from '@prisma/client';
import { invokeChaincode } from './utils/invoke';
import { asyncexecute } from './utils/shell';
import { v4 as uuidv4 } from 'uuid';
import { delay } from './utils/utils';

const router: Router = express.Router();
const prisma = new PrismaClient();

router.post('/create', async (req: Request, res: Response, next: NextFunction) => {
    const { userId } = req.body;
    try {
        if (!userId) throw new Error('userId is required');
        const findUser = await prisma.user.findUnique({
            where: {
                id: userId,
            },
        });
        if (!findUser) {
            throw new Error('User Not Found');
        } else if (findUser && findUser.wallet_addr) {
            throw new Error('Wallet already exists');
        } else if (findUser && findUser.wallet_addr === null) {
            const walletId = uuidv4();

            const createHFAccount = await invokeChaincode('CreateAccount', [userId]);
            console.log(createHFAccount);

            if (createHFAccount.success) {
                await delay(2000);
                const addFiat = await invokeChaincode('AddFiat', [userId, 10000]);
                console.log(addFiat);

                if (addFiat.success) {
                    const newWallet = await prisma.wallet.create({
                        data: {
                            addr: walletId,
                            fiat_balance: 10000,
                            owner_id: userId,
                        },
                    });
                    if (newWallet && newWallet.addr) {
                        const updatedUser = await prisma.user.update({
                            where: { id: userId },
                            data: { wallet_addr: newWallet.addr },
                        });
                        if (updatedUser) {
                            res.status(200).json({
                                message: 'success',
                                userId: userId,
                                walletId: walletId,
                            });
                        } else {
                            throw new Error('Update User Error');
                        }
                    } else {
                        throw new Error('Create Wallet Error');
                    }
                } else {
                    throw new Error('Add HF-Fiat Error');
                }
            } else {
                throw new Error('Create HF-Account Error');
            }
        }
    } catch (error) {
        console.error(error);
        res.status(500).json(error);
    }
});

export default router;