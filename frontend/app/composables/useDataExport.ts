/**
 * 数据导出 composable
 * 提供多种格式的数据导出功能
 */

export interface ExportOptions {
  filename?: string
  format?: 'json' | 'csv' | 'xlsx'
  includeHeaders?: boolean
  dateFormat?: string
}

export interface ExportColumn {
  key: string
  label: string
  formatter?: (value: any) => string
}

export const useDataExport = () => {
  const toast = useSmartToast()

  /**
   * 导出为 JSON 格式
   */
  const exportToJson = (data: any[], options: ExportOptions = {}) => {
    const { filename = 'data', dateFormat = 'YYYY-MM-DD' } = options
    
    try {
      const jsonData = JSON.stringify(data, null, 2)
      const blob = new Blob([jsonData], { type: 'application/json' })
      downloadBlob(blob, `${filename}-${formatDate(new Date(), dateFormat)}.json`)
      
      toast.success('JSON 文件导出成功')
    } catch (error) {
      toast.error('导出失败')
      console.error('JSON export error:', error)
    }
  }

  /**
   * 导出为 CSV 格式
   */
  const exportToCsv = (data: any[], columns: ExportColumn[], options: ExportOptions = {}) => {
    const { filename = 'data', includeHeaders = true, dateFormat = 'YYYY-MM-DD' } = options
    
    try {
      let csvContent = ''
      
      // 添加表头
      if (includeHeaders) {
        csvContent += columns.map(col => `"${col.label}"`).join(',') + '\n'
      }
      
      // 添加数据行
      data.forEach(row => {
        const values = columns.map(col => {
          let value = row[col.key]
          
          // 应用格式化函数
          if (col.formatter) {
            value = col.formatter(value)
          }
          
          // 处理特殊字符
          if (typeof value === 'string') {
            value = value.replace(/"/g, '""') // 转义双引号
          }
          
          return `"${value || ''}"`
        })
        
        csvContent += values.join(',') + '\n'
      })
      
      // 添加 BOM 以支持中文
      const bom = '\uFEFF'
      const blob = new Blob([bom + csvContent], { type: 'text/csv;charset=utf-8' })
      downloadBlob(blob, `${filename}-${formatDate(new Date(), dateFormat)}.csv`)
      
      toast.success('CSV 文件导出成功')
    } catch (error) {
      toast.error('导出失败')
      console.error('CSV export error:', error)
    }
  }

  /**
   * 导出为 Excel 格式（简化版）
   */
  const exportToExcel = (data: any[], columns: ExportColumn[], options: ExportOptions = {}) => {
    const { filename = 'data', dateFormat = 'YYYY-MM-DD' } = options
    
    try {
      // 创建简单的 XML 格式的 Excel 文件
      let xmlContent = `<?xml version="1.0"?>
<Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet"
 xmlns:o="urn:schemas-microsoft-com:office:office"
 xmlns:x="urn:schemas-microsoft-com:office:excel"
 xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet"
 xmlns:html="http://www.w3.org/TR/REC-html40">
 <Worksheet ss:Name="Sheet1">
  <Table>`

      // 添加表头
      xmlContent += '<Row>'
      columns.forEach(col => {
        xmlContent += `<Cell><Data ss:Type="String">${escapeXml(col.label)}</Data></Cell>`
      })
      xmlContent += '</Row>'
      
      // 添加数据行
      data.forEach(row => {
        xmlContent += '<Row>'
        columns.forEach(col => {
          let value = row[col.key]
          
          if (col.formatter) {
            value = col.formatter(value)
          }
          
          const dataType = typeof value === 'number' ? 'Number' : 'String'
          xmlContent += `<Cell><Data ss:Type="${dataType}">${escapeXml(String(value || ''))}</Data></Cell>`
        })
        xmlContent += '</Row>'
      })
      
      xmlContent += `  </Table>
 </Worksheet>
</Workbook>`
      
      const blob = new Blob([xmlContent], { type: 'application/vnd.ms-excel' })
      downloadBlob(blob, `${filename}-${formatDate(new Date(), dateFormat)}.xls`)
      
      toast.success('Excel 文件导出成功')
    } catch (error) {
      toast.error('导出失败')
      console.error('Excel export error:', error)
    }
  }

  /**
   * 通用导出函数
   */
  const exportData = (
    data: any[], 
    columns: ExportColumn[], 
    options: ExportOptions = {}
  ) => {
    const { format = 'csv' } = options
    
    if (!data || data.length === 0) {
      toast.warning('没有数据可导出')
      return
    }
    
    switch (format) {
      case 'json':
        exportToJson(data, options)
        break
      case 'csv':
        exportToCsv(data, columns, options)
        break
      case 'xlsx':
        exportToExcel(data, columns, options)
        break
      default:
        toast.error('不支持的导出格式')
    }
  }

  /**
   * 导出统计报告
   */
  const exportStatsReport = (stats: any[], options: ExportOptions = {}) => {
    const columns: ExportColumn[] = [
      { key: 'label', label: '指标名称' },
      { key: 'value', label: '数值', formatter: (v) => String(v) },
      { key: 'description', label: '描述' },
      { key: 'trend', label: '趋势', formatter: (v) => v === 'up' ? '上升' : v === 'down' ? '下降' : '稳定' }
    ]
    
    exportData(stats, columns, { 
      ...options, 
      filename: options.filename || 'stats-report' 
    })
  }

  /**
   * 导出用户列表
   */
  const exportUserList = (users: any[], options: ExportOptions = {}) => {
    const columns: ExportColumn[] = [
      { key: 'user_id', label: '用户ID' },
      { key: 'nickname', label: '昵称' },
      { key: 'email', label: '邮箱' },
      { key: 'user_type', label: '用户类型', formatter: (v) => v === 1 ? '临时用户' : '正式用户' },
      { key: 'created_at', label: '注册时间', formatter: (v) => formatDateTime(v) }
    ]
    
    exportData(users, columns, { 
      ...options, 
      filename: options.filename || 'user-list' 
    })
  }

  /**
   * 导出群组列表
   */
  const exportGroupList = (groups: any[], options: ExportOptions = {}) => {
    const columns: ExportColumn[] = [
      { key: 'group_id', label: '群组ID' },
      { key: 'name', label: '群组名称' },
      { key: 'description', label: '群组描述' },
      { key: 'member_count', label: '成员数量' },
      { key: 'creator_name', label: '创建者' },
      { key: 'created_at', label: '创建时间', formatter: (v) => formatDateTime(v) },
      { key: 'last_active_at', label: '最后活跃', formatter: (v) => formatDateTime(v) }
    ]
    
    exportData(groups, columns, { 
      ...options, 
      filename: options.filename || 'group-list' 
    })
  }

  // 辅助函数
  const downloadBlob = (blob: Blob, filename: string) => {
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.style.display = 'none'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  const formatDate = (date: Date, format: string) => {
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    
    return format
      .replace('YYYY', String(year))
      .replace('MM', month)
      .replace('DD', day)
  }

  const formatDateTime = (dateString: string) => {
    if (!dateString) return ''
    const date = new Date(dateString)
    return date.toLocaleString('zh-CN')
  }

  const escapeXml = (text: string) => {
    return text
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&apos;')
  }

  return {
    exportData,
    exportToJson,
    exportToCsv,
    exportToExcel,
    exportStatsReport,
    exportUserList,
    exportGroupList
  }
}
