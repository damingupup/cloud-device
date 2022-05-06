import copy
import logging
import traceback

import aiomysql
import aioredis

from aiomysql import DictCursor

from config import Config

log_obj = logging.getLogger('debug')


class DataTool(object):
    cur = None
    conn = None

    @staticmethod
    async def _pool():
        _pool = await aiomysql.create_pool(**Config.mysql_options)
        return _pool

    async def get_cursor(self):
        try:
            self.pool = await self._pool()
            self.conn = await self.pool.acquire()
            self.cur = await self.conn.cursor(DictCursor)
        except Exception as e:
            log_obj.error('数据库连接失败!!!')
            return False
        return True

    async def get_many(self, sql, param=None):
        res = False
        try:
            await self.cur.execute(sql, param)
            res = await self.cur.fetchall()
            if not res:
                res = []
            return res
        except:
            log_obj.error(traceback.format_exc())
        return res

    async def get_one(self, sql, param=None):
        try:
            await self.cur.execute(sql, param)
            result = await self.cur.fetchone()
            return result
        except:
            log_obj.error(traceback.format_exc())

    async def release(self):
        # 使用此函数释放连接
        if self.cur:
            await self.cur.close()
            self.conn.close()
        # 释放掉conn,将连接放回到连接池中
        try:
            await self.pool.release(self.conn)
            self.pool.close()
        except:
            log_obj.error('已经释放')
        try:
            self._redis_conn.close()
        except AttributeError:
            ...
        log_obj.error('释放掉pool')

    async def update_data(self, sql):
        # 更新数据,会返回id
        flag = await self.execute_sql(sql)
        return flag

    async def update_no_commit(self, sql, params=None, last_id=False, commit=False):
        flag = True
        try:
            await self.cur.execute(sql, params)
            if last_id:
                flag = self.cur.lastrowid
        except:
            flag = False
            await self.conn.rollback()
            log_obj.error("错误的sql:%s" % sql)
            log_obj.error(traceback.format_exc())
        if commit:
            await self.commit()
        return flag

    async def commit(self):
        flag = False
        try:
            await self.conn.commit()
            flag = True
        except:
            await self.conn.rollback()
            log_obj.error(traceback.format_exc())
        return flag

    async def execute_sql(self, sql, param=None):
        try:
            await self.cur.execute(sql, param)
            await self.conn.commit()
            flag = self.cur.lastrowid
        except Exception as e:
            flag = False
            await self.conn.rollback()
            log_obj.error(traceback.format_exc())
        return flag

    async def redis(self, single=False):
        host = Config.redis_options['host']
        _redis_conn = await aioredis.create_redis((host,6379))
        if not single:
            self._redis_conn = _redis_conn
        return _redis_conn

    async def rollback(self):
        try:
            await self.conn.rollback()
        except:
            return

    @staticmethod
    async def query(sql, param, all=False):
        _config = copy.deepcopy(Config.mysql_options)
        if _config.get('minsize', None):
            _config.pop('minsize')
        if _config.get('maxsize', None):
            _config.pop('maxsize')
        conn = await aiomysql.connect(**_config)
        cur = await conn.cursor(DictCursor)
        await cur.execute(sql, param)
        if not all:
            data = await cur.fetchone()
        else:
            data = await cur.fetchall()
        await cur.close()
        conn.close()
        return data
