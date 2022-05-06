# coding=utf-8
import smtplib
from email.mime.text import MIMEText
from email.utils import formataddr

from celery import Celery

from config import SENDER, MailPassword

backend = 'redis://127.0.0.1:6379/0'
broker = 'redis://127.0.0.1:6379/1'

app = Celery('tasks', backend=backend, broker=broker)


@app.task
def sendmail(recv, content, type_id):
    print(SENDER)
    print(recv)
    print(content)
    msg = MIMEText(content, _subtype='html', _charset='utf-8')
    msg['From'] = formataddr(("youkaQA", SENDER))  # 括号里的对应发件人邮箱昵称、发件人邮箱账号
    msg['To'] = formataddr((recv, recv))  # 括号里的对应收件人邮箱昵称、收件人邮箱账号
    if type_id == 1:
        msg['Subject'] = "云测平台账号注册"  # 邮件的主题
    elif type_id == 2:
        msg['Subject'] = "云测平台密码修改"  # 邮件的主题
    elif type_id == 3:
        msg['Subject'] = "云测平台邮箱修改"  # 邮件的主题
    else:
        msg['Subject'] = "云测平台账号注册"  # 邮件的主题
    smtp = smtplib.SMTP()
    smtp.connect("smtp.exmail.qq.com", 25)
    smtp.login(SENDER, MailPassword)  # 括号中对应的是发件人邮箱账号、邮箱密码
    smtp.sendmail(SENDER, [recv, ], msg.as_string())  # 括号中对应的是发件人邮箱账号、收件人邮箱账号、发送邮件
    smtp.quit()  # 关闭连接


app.autodiscover_tasks(['lib.tasks.sendmail'])
