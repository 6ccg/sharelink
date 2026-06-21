import type { Feature, FeatureCollection, Geometry } from 'geojson';
import { feature } from 'topojson-client';
import countries from 'i18n-iso-countries';
import worldAtlas from 'world-atlas/countries-110m.json';
import { ChinaData } from 'china-map-geojson';
import type { GeoItem } from './types';

export interface MapFeatureProps {
  name: string;
  type: 'country' | 'province';
  value: number;
}

export function buildMapFeatures(geo: GeoItem[]) {
  const countryPV = new Map<string, number>();
  const provincePV = new Map<string, number>();

  for (const item of geo) {
    const country = item.country || '';
    if (!country || country === '内网/本地' || country === '未知') continue;

    if (country === '中国') {
      const province = normalizeChinaProvince(item.region);
      if (province) provincePV.set(province, (provincePV.get(province) || 0) + item.pv);
      continue;
    }

    const iso = COUNTRY_NAME_TO_ISO[country];
    const numeric = iso ? countries.alpha2ToNumeric(iso) : undefined;
    if (numeric) countryPV.set(numeric, (countryPV.get(numeric) || 0) + item.pv);
  }

  const world = feature(
    worldAtlas as unknown as Parameters<typeof feature>[0],
    (worldAtlas as unknown as { objects: { countries: Parameters<typeof feature>[1] } }).objects.countries
  ) as unknown as FeatureCollection<Geometry, { name: string }>;

  const countryFeatures = world.features
    .filter((item) => String(item.id) !== '156')
    .map((item) => withMapProps(item, {
      name: item.properties?.name || String(item.id || ''),
      type: 'country',
      value: countryPV.get(String(item.id).padStart(3, '0')) || 0,
    }));

  const provinceFeatures = (ChinaData as FeatureCollection<Geometry, { name: string }>).features.map((item) =>
    withMapProps(item, {
      name: item.properties?.name || '',
      type: 'province',
      value: provincePV.get(normalizeChinaProvince(item.properties?.name || '')) || 0,
    })
  );

  const features = [...countryFeatures, ...provinceFeatures];
  const maxPV = features.reduce((max, item) => Math.max(max, item.properties.value), 0);
  return { features, maxPV };
}

function withMapProps(
  item: Feature<Geometry, unknown>,
  properties: MapFeatureProps
): Feature<Geometry, MapFeatureProps> {
  return {
    ...item,
    properties,
  };
}

function normalizeChinaProvince(region: string) {
  return region
    .replace(/省|市|壮族自治区|回族自治区|维吾尔自治区|自治区|特别行政区/g, '')
    .trim();
}

const COUNTRY_NAME_TO_ISO: Record<string, string> = {
  '美国': 'US', '日本': 'JP', '韩国': 'KR', '英国': 'GB',
  '法国': 'FR', '德国': 'DE', '加拿大': 'CA', '澳大利亚': 'AU', '俄罗斯': 'RU',
  '巴西': 'BR', '印度': 'IN', '意大利': 'IT', '西班牙': 'ES', '墨西哥': 'MX',
  '印度尼西亚': 'ID', '土耳其': 'TR', '沙特阿拉伯': 'SA', '荷兰': 'NL', '瑞士': 'CH',
  '阿根廷': 'AR', '瑞典': 'SE', '波兰': 'PL', '比利时': 'BE', '泰国': 'TH',
  '奥地利': 'AT', '挪威': 'NO', '以色列': 'IL', '爱尔兰': 'IE', '新加坡': 'SG',
  '马来西亚': 'MY', '菲律宾': 'PH', '丹麦': 'DK', '芬兰': 'FI', '智利': 'CL',
  '哥伦比亚': 'CO', '南非': 'ZA', '埃及': 'EG', '葡萄牙': 'PT', '捷克': 'CZ',
  '罗马尼亚': 'RO', '新西兰': 'NZ', '越南': 'VN', '希腊': 'GR', '伊拉克': 'IQ',
  '秘鲁': 'PE', '乌克兰': 'UA', '匈牙利': 'HU', '阿联酋': 'AE', '孟加拉': 'BD',
  '巴基斯坦': 'PK', '尼日利亚': 'NG', '缅甸': 'MM', '斯里兰卡': 'LK', '柬埔寨': 'KH',
  '老挝': 'LA', '蒙古': 'MN', '朝鲜': 'KP', '尼泊尔': 'NP', '不丹': 'BT',
  '阿富汗': 'AF', '伊朗': 'IR', '哈萨克斯坦': 'KZ', '乌兹别克斯坦': 'UZ',
  '土库曼斯坦': 'TM', '塔吉克斯坦': 'TJ', '吉尔吉斯斯坦': 'KG', '格鲁吉亚': 'GE',
  '亚美尼亚': 'AM', '阿塞拜疆': 'AZ', '叙利亚': 'SY', '约旦': 'JO', '黎巴嫩': 'LB',
  '科威特': 'KW', '卡塔尔': 'QA', '巴林': 'BH', '阿曼': 'OM', '也门': 'YE',
  '冰岛': 'IS', '卢森堡': 'LU', '保加利亚': 'BG', '克罗地亚': 'HR', '斯洛伐克': 'SK',
  '斯洛文尼亚': 'SI', '立陶宛': 'LT', '拉脱维亚': 'LV', '爱沙尼亚': 'EE',
  '塞尔维亚': 'RS', '白俄罗斯': 'BY', '摩尔多瓦': 'MD', '北马其顿': 'MK',
  '阿尔巴尼亚': 'AL', '波黑': 'BA', '黑山': 'ME', '科索沃': 'XK',
  '肯尼亚': 'KE', '坦桑尼亚': 'TZ', '乌干达': 'UG', '埃塞俄比亚': 'ET',
  '加纳': 'GH', '喀麦隆': 'CM', '摩洛哥': 'MA', '突尼斯': 'TN', '阿尔及利亚': 'DZ',
  '利比亚': 'LY', '苏丹': 'SD', '刚果': 'CG', '安哥拉': 'AO', '莫桑比克': 'MZ',
  '赞比亚': 'ZM', '津巴布韦': 'ZW', '博茨瓦纳': 'BW', '纳米比亚': 'NA',
  '塞内加尔': 'SN', '马里': 'ML', '尼日尔': 'NE', '乍得': 'TD', '索马里': 'SO',
  '马达加斯加': 'MG', '卢旺达': 'RW', '布隆迪': 'BI',
  '委内瑞拉': 'VE', '厄瓜多尔': 'EC', '玻利维亚': 'BO', '巴拉圭': 'PY', '乌拉圭': 'UY',
  '古巴': 'CU', '海地': 'HT', '多米尼加': 'DO', '巴拿马': 'PA', '哥斯达黎加': 'CR',
  '危地马拉': 'GT', '洪都拉斯': 'HN', '萨尔瓦多': 'SV', '尼加拉瓜': 'NI',
  '中国台湾': 'TW', '中国香港': 'HK', '中国澳门': 'MO',
};
