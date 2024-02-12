<template>
  <div class="header">
    <div class="left">
      <p>UID: {{ userInfo.MID }}</p>
      <p>U P: {{ userInfo.BName }}</p>
      <p>本地累计粉丝数: {{ userInfo.MidFansNum }}</p>
    </div>
    <div class="right">
      <img src="./assets/wx.png" alt="微信关注【逗之助的窝】" width="500">
    </div>
  </div>
  <hr>

  <div>
    <table>
      <thead>
        <tr>
          <th style="width: 40%">关键字（使用|隔开）</th>
          <th style="width: 50%">回复内容</th>
          <th style="width: 10%">操作</th>
        </tr>
      </thead>

      <tbody>
        <tr>
          <td><textarea name="content" v-model="Keys"></textarea></td>
          <td><textarea name="content" v-model="ReplyMsg"></textarea></td>
          <td>
            <button type="button" @click="Add">添加</button>
          </td>
        </tr>
        <tr>
          <td>未关注用户回复内容（关键词触发，留空表示禁用）</td>
          <td><textarea name="content" v-model="userInfo.ReplyMsgUnFollow"></textarea></td>
          <td>
            <button type="button" @click="AddUnFollower">保存</button>
          </td>
        </tr>
        <tr v-for=" item in userInfo.ReplyMsgList" :key="item.ID">
          <td><textarea v-model="item.Keys"></textarea></td>
          <td><textarea v-model="item.Content"></textarea></td>
          <td>
            <button type="button" @click="Save(item)">保存</button>
            --
            <button type="button" @click="Del(item.ID)">删除</button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script>
import axios from 'axios';

const base_url = "http://127.0.0.1:9558/api"


export default {

  data() {
    return {
      Keys: "",
      ReplyMsg: "",
      userInfo: {},
    }
  },
  mounted() {
    axios.get(base_url + '/index')
      .then(response => {
        // 请求成功时更新数据
        console.log(response.data);

        let data = response.data.userInfo;
        if (data.ReplyMsgList != null) {
          for (let i = 0; i < data.ReplyMsgList.length; i++) {
            const element = data.ReplyMsgList[i];
            element.Keys = element.Keys.join("|")
          }
        }
        this.userInfo = data
      }).catch(error => {
        // 请求失败时处理错误
        console.error('Error fetching data:', error);
      })
  },
  computed: {},
  methods: {
    Add() {
      if (this.Keys != "" && this.ReplyMsg != "") {
        axios.post(base_url + '/add', {
          keys: this.Keys,
          replyMsg: this.ReplyMsg
        }).then(response => {
          if (response.data.code == 'ok') {
            location.reload()
          }
        }).catch(error => {
          console.error('Error fetching data:', error);
        })
      }
    },
    AddUnFollower() {
      axios.post(base_url + '/unfollower', {
        ReplyMsgUnFollow: this.userInfo.ReplyMsgUnFollow
      }).then(response => {
        if (response.data.code == 'ok') {
          location.reload()
        }
      }).catch(error => {
        console.error('Error fetching data:', error);
      })
    },
    Save(item) {
      console.log(item);
      axios.post(base_url + '/save', {
        ID: item.ID,
        keys: item.Keys.split("|"),
        content: item.Content
      }).then(response => {
        if (response.data.code == 'ok') {
          location.reload()
        }
      }).catch(error => {
        console.error('Error fetching data:', error);
      })
    },
    Del(id) {
      axios.get(base_url + '/del', {
        params: {
          id: id
        }
      }).then(response => {
        if (response.data.code == 'ok') {
          location.reload()
        }
      }).catch(error => {
        console.error('Error fetching data:', error);
      })
    }
  },
}

</script>


<style>
table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 20px;
}

th,
td {
  border: 1px solid #ddd;
  padding: 8px;
  text-align: left;
}

th {
  background-color: #f2f2f2;
}

textarea {
  width: 100%;
  height: 100%;
}

.header {
  display: flex;
  flex-direction: row;
  justify-content: space-between;
  align-items: center;
}
</style>