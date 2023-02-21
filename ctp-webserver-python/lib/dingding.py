import datetime

import pymysql
import requests
import time
import hmac
import hashlib
import base64
import urllib.parse

from pymysql.cursors import DictCursor

from config import DING_DING_SECRET, DING_DING_URL, Config


class DingDingRoot(object):
    @staticmethod
    def get_sign(secret):
        timestamp = str(round(time.time() * 1000))
        secret_enc = secret.encode('utf-8')
        string_to_sign = '{}\n{}'.format(timestamp, secret)
        string_to_sign_enc = string_to_sign.encode('utf-8')
        hmac_code = hmac.new(secret_enc, string_to_sign_enc, digestmod=hashlib.sha256).digest()
        sign = urllib.parse.quote_plus(base64.b64encode(hmac_code))
        return sign, timestamp
    def query(self):
        sql = 'select br.deviceId,ttd.name,tu.email,tu.phone,br.borrowTime,ttd.borrow_status,tu.name as username,' \
              'br.statusId from borrowing_record  br left join ' \
              'tb_device ttd on br.deviceId = ttd.id left join tb_user tu on br.userId=tu.id where ' \
              'br.isDelete=false and ttd.borrow_status !=0; '
        num = self.cur.execute(sql)
        resp = self.cur.fetchall()
        return resp

    def send(self, msg):
        import logging
        log_obj = logging.getLogger("debug")
        secret = DING_DING_SECRET
        ding_url = DING_DING_URL
        log_obj.info(secret)
        log_obj.info(ding_url)
        if datetime.datetime.now().hour < 10:
            secret = "SEC7e1a0ecf88b91eda686e2f19c95069847c77b2f5d5740061374e609fa0438fad"
            ding_url = 'https://oapi.dingtalk.com/robot/send?access_token=8e39ed58be429f880dda99d1cdc54034eb3fbf2643759061d3c78c0bddf0b1b9'
        log_obj.info(secret)
        log_obj.info(ding_url)
        sign, timestamp = self.get_sign(secret)
        url = ding_url + '&timestamp={}&sign={}'.format(timestamp, sign)
        data = requests.post(url, json=msg)
        print(data.json())
        log_obj.info(data.json())


class DataHandle(object):
    def __init__(self):
        self.conn = self._get_connect()
        self.cur = self.conn.cursor(DictCursor)

    @staticmethod
    def _get_connect():
        _conn = pymysql.connect(**Config.mysql_options)
        return _conn

    def query(self):
        sql = 'select br.deviceId,ttd.name,tu.email,tu.phone,br.borrowTime,ttd.borrow_status,tu.name as username,' \
              'br.statusId from borrowing_record  br left join ' \
              'tb_device ttd on br.deviceId = ttd.id left join tb_user tu on br.userId=tu.id where ' \
              'br.isDelete=false and ttd.borrow_status !=0; '
        num = self.cur.execute(sql)
        resp = self.cur.fetchall()
        return resp


def pushMessage():
    data = DataHandle()
    return_resp = data.query()
    return_phone_num = []
    return_message = []
    borrow_phone_num = []
    borrow_message = []
    today = datetime.datetime.now()
    month = 30
    monday = 0
    for i in return_resp:
        borrow_time = i['borrowTime']
        if i['borrow_status'] == 2 and i['statusId'] == 3:
            if today.date() == borrow_time.date():
                borrow_phone_num.append(i['phone'])
                msg = "请{username}领取手机{name}".format(username=i['username'], name=i['name'])
                borrow_message.append(msg)
                continue
            if borrow_time.isoweekday() == 1:
                continue
            if i['username'] == '李赛威':
                continue
            if (today - borrow_time).days > 3:
                if (today - borrow_time).days > month and today.weekday() == monday:
                    # 超过一个月 每周一提醒
                    return_phone_num.append(i['phone'])
                    msg = "请{username}归还手机{name} 借用日期{borrow_time}".format(username=i['username'], name=i['name'],
                                                                           borrow_time=borrow_time)
                    return_message.append(msg)
                continue
            return_phone_num.append(i['phone'])
            msg = "请{username}归还手机{name} 借用日期{borrow_time}".format(username=i['username'], name=i['name'],
                                                                   borrow_time=borrow_time)
            return_message.append(msg)
        elif i['borrow_status'] == 1:
            if (today - borrow_time).days > 3:
                continue
            borrow_phone_num.append(i['phone'])
            msg = "请{username}领取手机{name}".format(username=i['username'], name=i['name'])
            borrow_message.append(msg)
    return_message = '\n'.join(return_message)
    borrow_message = '\n'.join(borrow_message)
    ding = DingDingRoot()
    borrow_data = {
        "msgtype": "text",
        "text": {
            "content": borrow_message
        },
        "at": {
            "atMobiles": borrow_phone_num,
            "isAtAll": False
        }
    }
    return_data = {
        "msgtype": "text",
        "text": {
            "content": return_message
        },
        "at": {
            "atMobiles": return_phone_num,
            "isAtAll": False
        }
    }
    today = datetime.datetime.now()
    print(borrow_data)
    ding.send(borrow_data)
    if today.hour < 12:
        ding.send(return_data)


if __name__ == '__main__':
    pushMessage()
