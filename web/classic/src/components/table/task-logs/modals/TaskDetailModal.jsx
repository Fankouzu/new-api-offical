/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import { Button, Descriptions, Modal, Space, Tag, Typography } from '@douyinfe/semi-ui';
import { IconCode, IconEyeOpened } from '@douyinfe/semi-icons';
import { timestamp2string } from '../../../../helpers';

const { Text } = Typography;

function formatTime(value) {
  return value ? timestamp2string(value) : '-';
}

function formatBytes(value) {
  const bytes = Number(value || 0);
  if (!Number.isFinite(bytes) || bytes <= 0) return '-';
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(2)} KB`;
  return `${bytes} B`;
}

function renderText(value) {
  if (value === undefined || value === null || value === '') return '-';
  return String(value);
}

function resultTypeLabel(type, t) {
  switch (type) {
    case 'image':
      return t('图片');
    case 'video':
      return t('视频');
    case 'audio':
      return t('音频');
    default:
      return t('未知');
  }
}

function resultTag(result, t) {
  if (!result?.available) {
    return <Tag color='grey'>{t('无')}</Tag>;
  }
  const color = result.type === 'image' ? 'cyan' : result.type === 'video' ? 'blue' : 'grey';
  return (
    <Space spacing={6}>
      <Tag color={color}>{resultTypeLabel(result.type, t)}</Tag>
      {result.inline ? <Tag color='orange'>{t('内联媒体')}</Tag> : null}
    </Space>
  );
}

const TaskDetailModal = ({
  t,
  taskDetail,
  isTaskDetailOpen,
  setIsTaskDetailOpen,
  openTaskRawModal,
  openVideoModal,
  openImageModal,
  openContentModal,
}) => {
  const detail = taskDetail || {};
  const result = detail.result || {};
  const dataSummary = detail.data_summary || {};

  const rows = [
    { key: t('ID'), value: renderText(detail.id) },
    { key: t('任务ID'), value: renderText(detail.task_id) },
    { key: t('平台'), value: renderText(detail.platform) },
    { key: t('类型'), value: renderText(detail.action) },
    { key: t('状态'), value: renderText(detail.status) },
    { key: t('进度'), value: renderText(detail.progress) },
    { key: t('渠道'), value: renderText(detail.channel_id) },
    { key: t('用户'), value: renderText(detail.username || detail.user_id) },
    { key: t('提交时间'), value: formatTime(detail.submit_time) },
    { key: t('开始时间'), value: formatTime(detail.start_time) },
    { key: t('完成时间'), value: formatTime(detail.finish_time) },
    { key: t('结果'), value: resultTag(result, t) },
    { key: t('结果大小'), value: result.size ? formatBytes(result.size) : '-' },
    { key: t('请求数据大小'), value: formatBytes(dataSummary.bytes) },
  ];

  if (detail.fail_reason) {
    rows.push({
      key: t('详情'),
      value: (
        <Text ellipsis={{ rows: 3, showTooltip: true }}>
          {detail.fail_reason}
        </Text>
      ),
    });
  }

  const previewResult = () => {
    if (!result?.url) return;
    if (result.type === 'image') {
      openImageModal(result.url);
      return;
    }
    if (result.type === 'video') {
      openVideoModal(result.url);
      return;
    }
    openContentModal(result.url);
  };

  return (
    <Modal
      title={t('任务详情')}
      visible={isTaskDetailOpen}
      onCancel={() => setIsTaskDetailOpen(false)}
      footer={null}
      centered
      closable
      width={720}
      bodyStyle={{ maxHeight: '70vh', overflow: 'auto', padding: 16 }}
    >
      <Descriptions data={rows} />
      <Space style={{ marginTop: 16 }} wrap>
        {result?.available && result?.url ? (
          <Button icon={<IconEyeOpened />} onClick={previewResult}>
            {t('预览结果')}
          </Button>
        ) : null}
        {detail?.id ? (
          <Button
            type='tertiary'
            icon={<IconCode />}
            onClick={() => openTaskRawModal(detail.id)}
          >
            {t('查看原始数据')}
          </Button>
        ) : null}
      </Space>
    </Modal>
  );
};

export default TaskDetailModal;
