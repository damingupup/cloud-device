import base64
import json
import rsa

from main import public_file, private_file


class TokenInfo(object):
    @classmethod
    def get_encrypt(cls, info):
        '''
            公钥加密，返回密文，顺便可以保存到文件
        '''
        info = info.encode()
        pubkey = rsa.PublicKey.load_pkcs1(public_file)
        s_encrypt = rsa.encrypt(info, pubkey)
        s_encrypt = base64.b64encode(s_encrypt).decode()
        return s_encrypt

    @classmethod
    def get_decrypt(cls, info):
        '''
            私钥解密，返回明文
        '''
        try:
            info = base64.b64decode(info.encode())
            prikey = rsa.PrivateKey.load_pkcs1(private_file)
            s_decrypt = rsa.decrypt(info, prikey).decode()
        except:
            return False
        return eval(s_decrypt)

    @staticmethod
    def makekey():
        # 创建公钥与私钥的方法
        (pubkey, privkey) = rsa.newkeys(1024)
        pub = pubkey.save_pkcs1()
        pubfile = open('public1.pem', 'wb')
        pubfile.write(pub)
        pubfile.close()
        pri = privkey.save_pkcs1()
        prifile = open('private1.pem', 'wb')
        prifile.write(pri)
        prifile.close()


if __name__ == '__main__':
    TokenInfo.makekey()
