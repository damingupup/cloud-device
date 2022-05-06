import hashlib
import json
import re
from datetime import datetime
from typing import Union

import tornado.web
import tornado.ioloop
from dateutil import parser
from dateutil.parser import ParserError
from tornado.httpclient import AsyncHTTPClient

from lib.driver import DataTool
from lib.make_token import TokenInfo


class Tool(DataTool):
    def __init__(self):
        self.get_args = {}
        self.post_args = {}
        self.put_args = {}
        self.delete_args = {}

    @staticmethod
    def validate_email(email) -> bool:
        res = re.match(
            r"[\w!#$%&'*+/=?^_`{|}~-]+(?:\.[\w!#$%&'*+/=?^_`{|}~-]+)*@(?:[\w](?:[\w-]*[\w])?\.)+[\w](?:[\w-]*[\w])?",
            email)
        if res is not None:
            return True
        else:
            return False

    @staticmethod
    def password_hash(password):
        password_md5 = hashlib.md5(password.encode()).hexdigest()
        return password_md5


class CorsMixin(tornado.web.RequestHandler):
    CORS_ORIGIN = '*'
    CORS_METHODS = 'GET,POST,OPTIONS,PUT,DELETE'
    CORS_CREDENTIALS = True
    CORS_HEADERS = "x-requested-with,authorization"

    def set_default_headers(self):
        self.set_header("Access-Control-Allow-Origin", self.CORS_ORIGIN)
        self.set_header("Access-Control-Allow-Headers", self.CORS_HEADERS)
        self.set_header('Access-Control-Allow-Methods', self.CORS_METHODS)

    def options(self, *args):
        self.set_status(204)
        self.finish()


class BaseObj(CorsMixin, Tool):
    # 长度判断为low<=field<=high
    # default被设置为参数验证的默认值，参数命名注意避免占用
    Right_code = 0
    Error_Param__code = 1  # 参数错误
    Error_Auth__code = 2  # 认证错误
    Error_SERVER__code = 3  # 服务错误
    policy_list = {0: '普通用户', 1: '超级管理员', 2: '其他'}
    role_list = {0: '测试', 1: '开发', 2: '其他'}
    errorMsg = ''
    method_args = {"GET": "get_args", "POST": "post_args", "PUT": "put_args", "DELETE": "delete_args"}
    state_list = {0: '空闲', 1: '占用', 2: '离线', 3: "损坏", 4: "调试", 5: "释放中", 6: "租借中"}
    OnLine = '1'  # 设备已连接
    OffLine = '0'  # 设备离线
    FULL_LOAD = 2
    HIGH = 1
    LOW = 0

    @property
    def logger(self):
        log_obj = tornado.ioloop.IOLoop.current().log_obj
        return log_obj

    def initialize(self):
        self.get_args = dict()
        self.post_args = dict()
        self.put_args = dict()
        self.delete_args = dict()

    async def get_cour(self):
        self._redis_conn = None
        try:
            conn = await self.get_cursor()
            if not conn:  # 获取数据库连接状态
                self.errorMsg.append('数据库连接失败')
        except Exception as e:
            self.logger.error(msg=str(e))
            return self.errorMsg.append('数据库连接失败')
        return conn

    def on_finish(self) -> None:
        # 释放连接
        try:
            tornado.ioloop.IOLoop.current().spawn_callback(self.release)
        except Exception as e:
            self.logger.warning(str(e))
            self.logger.warning('数据库连接释放失败')

    async def prepare(self):
        self.index = 0
        self.logger.info('请求参数:{}'.format(self.request.body.decode()))
        header = self.request.headers
        # ws标志

        websocket = 'websocket'
        if websocket != header.get("Upgrade"):
            # 不为websocket请求时获取连接
            conn = await self.get_cour()
            if not conn:  # 获取数据库连接状态
                self.errorMsg.append('数据库连接失败')
                return
        self.flag = True
        self.errorMsg = []
        self.params = {}
        method = self.request.method
        try:
            method = self.method_args[method]
            rules = getattr(self, method)
        except Exception as e:
            self.logger.warning(str(e))
            return
        for field, rule in rules.items():
            if field == 'default':  # 跳过默认值
                continue
            # 获取规则
            null = rule.get('null', True)
            low, high = rule.get('long', (0, 0))
            field_type = rule.get('type', str)
            # 获取参数首先从查询参数或者form-data中获取，如果请求类型为post或者put会从json中获取
            param = self.get_param(field)
            # 判空验证
            if not param or param == 'null':
                if not null and param != 0:
                    self.errorMsg.append({field: '不能为空'})
                    self.flag = False
                continue
            # 长度验证
            if type(param) is int:
                param_length = len(str(param))
                if not low <= param_length <= high and (low != high != 0):
                    self.errorMsg.append({field: '大小需要在{}～{}之间'.format(low, high)})
                    self.flag = False
                    continue
            elif type(param) is str:
                long_param = len(param)
                if not low <= long_param <= high and (low != high != 0):
                    self.errorMsg.append({field: '长度需要在{}～{}之间'.format(low, high)})
                    self.flag = False
                    continue
            # 类型验证
            if not field_type:
                continue
            if field_type in (int, dict, list):  # 兼容数据类型
                try:
                    type_param = type(eval(str(param)))
                except:
                    self.errorMsg.append({field: '数据类型应该为{}'.format(str(field_type.__name__))})
                    continue
            else:  # str类型不需要转换
                type_param = type(param)
            if type_param != field_type:
                self.errorMsg.append({field: '数据类型应该为{}'.format(str(field_type.__name__))})
                self.flag = False
                continue

    def jscode(self, code=Right_code, msg='', data='', total=0):
        if type(data) == list and not total:
            total = len(data)
        if type(msg) != str:
            msg = str(msg)
        if code == self.Error_Auth__code:  # 认证失败
            if not msg:
                self.set_status(401)
                data = {'cd': code, 'msg': '用户未登陆或权限不足', 'data': data, 'total': total}
            else:
                data = {'cd': code, 'msg': msg, 'data': data, 'total': total}
        else:
            data = {'cd': code, 'msg': msg, 'data': data, 'total': total}
        return self.write(data)

    async def ws_message(self, code=Right_code, msg='', data='', cmd="device"):
        total = 0
        if type(data) == list:
            total = len(data)
        if type(msg) != str:
            msg = str(msg)
        data = {'cd': code, 'msg': msg, 'data': data, 'total': total}
        if cmd != "":
            data["cmd"] = cmd
        data = json.dumps(data)
        await self.write_message(data)
        if code != self.Right_code:
            self.logger.info(msg='我自由了')
            self.logger.info(msg=msg)
            self.close()

    def _request_summary(self):
        return "%s %s (%s) %s" % (self.request.method, self.request.uri,
                                  self.request.remote_ip, self.request.arguments)

    def get_param(self, name):
        json_data = self.request.body.decode()
        try:
            json_data = json.loads(json_data)
        except:
            json_data = dict()
        filed = json_data.get(name, 'null')
        if filed != 'null':
            return filed
        filed = self.get_argument(name, strip=True, default='')
        if filed == 'null':
            filed = None
        return filed

    async def get_current_user(self, policy_id: list = 0) -> Union[bool, dict]:
        if not policy_id:
            policy_id = []
        token = self.request.headers.get('token')
        if not token:
            # header中没有token再去请求参数中获取
            token = self.get_param('token')
        if not token:
            return False
        flag = TokenInfo.get_decrypt(token)
        if not flag:
            self.logger.info('token验证错误')
            return False
        policy = flag['policy']
        if policy_id and policy not in policy_id:  # 判断权限，之后可以将policyId改为list
            return False
        user_id = flag['id']
        user_sql = '''select id,policy,name,email,avatarId from tb_user 
        where id = %s  and isDelete = 0 LIMIT 1;'''
        res = await self.get_one(user_sql, user_id)
        if not res:
            return False
        return res

    @staticmethod
    def list_tuple(params: list):
        res = str(tuple(set(params)))
        if res[-2] == ',':
            res = res[:-2] + res[-1]
        return res

    async def get_policy(self, flag, group_id):
        policy = flag['policy']
        user_id = flag['id']
        if policy == 1:
            return True
        policy_sql = '''select tgu.id from tb_group_user tgu left join tb_user as tu where 
        tu.isDelete = 0 and tgu.isDelete = 0 and tgu.isAdmin = 1 and tgu.groupId = {groupId} and 
        tgu.userId = {userId} and tu.isDelete=0;'''.format(groupId=group_id, userId=user_id)
        res = await self.get_many(policy_sql)
        if not res:
            return False
        return True

    async def exist_group(self, group: list):  # 判断组是否存在
        num = len(set(group))
        group_str = self.list_tuple(group)
        exist_sql = '''select id from tb_group where id in 
        {group_str} and isDelete = 0;'''.format(group_str=group_str)
        res = await self.get_many(exist_sql)
        if not res:
            return False
        if len(res) != num:
            return False
        return True

    async def exist_user(self, user: list):  # 判读用户是否存在
        num = len(set(user))
        user_str = self.list_tuple(user)
        exist_sql = '''select id from tb_user where id in 
        {user_str} and isDelete = 0;'''.format(user_str=user_str)
        res = await self.get_many(exist_sql)
        if not res:
            return False
        if len(res) != num:
            return False
        return True

    async def exist_email(self, email: list):
        # 判读邮箱是否存在
        num = len(set(email))
        email = self.list_tuple(email)
        exist_sql = '''select id from tb_user where email in 
        {email} and isDelete = 0;'''.format(email=email)
        res = await self.get_many(exist_sql)
        if not res:
            return False
        if len(res) != num:
            return False
        return True

    @staticmethod
    def time_to_str(res: list):  # 时间格式
        for i in res:
            for k, v in i.items():
                if k in ('createtime', 'updatetime', 'borrowTime'):
                    i[k] = str(v)
        return res

    async def get_user_info(self, res):  # 获取人员信息
        if not res:
            return []
        group_id = self.list_tuple(list({i['id'] for i in res}))
        user_sql = '''select tgu.userId,tu.name,tu.email,tgu.isAdmin,tgu.groupId,tu.createtime from 
                            tb_group_user as tgu left join tb_user as tu on tu.id=tgu.userId where 
                            tgu.isDelete=0 and tu.isDelete=0 and tgu.isAdmin !=2 and tgu.groupId in 
                            {groupId};'''.format(groupId=group_id)
        user_res = await self.get_many(user_sql)
        user_res = self.time_to_str(user_res)
        for i in res:
            group = i['id']
            i['admin'] = []
            i['user'] = []
            user_list = []
            for user in user_res:
                if user.get("userId"):
                    user['id'] = user.pop('userId')
                if user['groupId'] == group and user['id'] not in user_list:
                    user_list.append(user['id'])
                    if user['isAdmin'] == 1:
                        i['admin'].append(user)
                    i['user'].append(user)
        return res

    async def _get_user_group(self, group_sql):
        res = await self.get_many(group_sql)
        if not res:
            return []
        res = {i['id']: i for i in res}
        res = [i for i in res.values()]
        res = self.time_to_str(res)
        res = await self.get_user_info(res)
        return res

    async def get_group_policy(self, user_id):  # 获取用户为管理员的所有的组
        sql = '''select tgu.groupId from tb_user as tu inner join tb_group_user as tgu on tu.id = tgu.userId
        and tu.isDelete=0 and tgu.isDelete=0 and tgu.isAdmin = 1 and tu.id = {userId};'''.format(userId=user_id)
        res = await self.get_many(sql)
        if not res:
            return []
        policy_id = [int(i['groupId']) for i in res]
        return policy_id

    async def get_group_user(self, user_id, group_id) -> bool:
        # 判读用户是否在组内
        sql = '''select id from tb_group_user where userId = %s and groupId = %s and isDelete = 0;'''
        res = await self.get_many(sql, [user_id, group_id])
        if not res:
            return False
        return True

    async def del_file(self, info: list):
        md5_id = [i['md5Id'] for i in info]
        if not md5_id:
            return
        md5_id_str = self.list_tuple(md5_id)
        self.logger.info(md5_id_str)
        file_sql = '''SELECT md5Id ,COUNT(md5Id) as num FROM tb_file WHERE isDelete=FALSE and md5Id in (%s) GROUP BY md5Id ;'''
        resp = await self.get_many(file_sql, md5_id_str[1:-1])
        del_md5_id = [i['md5Id'] for i in resp if i['num'] == 1]
        self.logger.info(del_md5_id)
        client = AsyncHTTPClient()
        res = {"success": [], "error": []}
        for file in info:
            domain = file['domain']
            path = file['path']
            file_id = file['id']
            md5_id = file['md5Id']
            if md5_id not in del_md5_id:
                res['success'].append(file_id)
                continue
            try:
                group = path.split('/')[1]
                url = '''{domain}/{group}/delete?md5={md5_id}'''.format(domain=domain, group=group, md5_id=md5_id)
                del_res = await client.fetch(url)
                del_res = json.loads(del_res.body.decode())
                if del_res['status'] != 'ok':
                    res['error'].append(file_id)
                    continue
                res['success'].append(file_id)
            except:
                res['error'].append(file_id)
        return res

    def close_redis(self, redis_conn):
        try:
            redis_conn.close()
        except:
            pass

    @staticmethod
    def get_time(st, et):
        try:
            if st != "":
                st = parser.parse(st)
            else:
                st = "20200101"
                st = parser.parse(st)
            if et != "":
                et = parser.parse(et)
            else:
                et = datetime.now()
        except ParserError:
            st = "20200101"
            st = parser.parse(st)
            et = datetime.now()
        st = st.strftime("%Y-%m-%d")
        et = et.strftime("%Y-%m-%d")
        return st, et

    def judge_status(self, data):
        for i in data:
            if i['time'] >= 7 * 60 * 60:
                i['high'] = self.FULL_LOAD
            elif 7 * 60 * 60 > i['time'] >= 4 * 60 * 60:
                i['high'] = self.HIGH
            else:
                i['high'] = self.LOW
            i['time'] = str(i["time"])
        return data
