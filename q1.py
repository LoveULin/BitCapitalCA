
import pathlib
import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt
import itertools
import scipy
from pandas.plotting import register_matplotlib_converters
register_matplotlib_converters()

_data_path = pathlib.PurePath() / 'bitcapital'
_data_file_list = ['Kraken_BTCUSD_1h.csv', 'Kraken_ETHUSD_1h.csv',
                   'Kraken_XRPUSD_1h.csv', 'Kraken_LTCUSD_1h.csv']


class Solution:
    def quiz1(self):
        '''retrieve BTC, ETH, XRP, LTC historical price data from KRAKEN, plot the trend base on them and
        calculate their Pearson correlation coefficient'''
        _, ax = plt.subplots(len(_data_file_list), figsize=(12, 7))

        datas = {}
        for i, file in enumerate(_data_file_list):
            pair_name = file.split('_')[1]
            pair_data = pd.read_csv(
                _data_path / file, index_col='Date', parse_dates=True, skiprows=1)
            pair_data.set_index(pd.to_datetime(
                pair_data.index, format='%Y-%m-%d %I-%p'), inplace=True, verify_integrity=True)
            datas[pair_name] = pair_data.sort_index()
            print('pair: {}, head data: \n{}'.format(
                pair_name, datas[pair_name].head(10)))
            new_ylabel = pair_name + '($)'
            g = sns.relplot(x='Date', y=new_ylabel, kind='line',
                            data=datas[pair_name].head(50).rename(columns={'Close': new_ylabel}).reset_index(), ax=ax[i])
            g.set(ylabel=pair_name)
            plt.close(g.fig)

        plt.tight_layout()
        plt.show()

        for c1, c2 in itertools.combinations(datas.keys(), 2):
            print('Pearson correlation coefficient between {} and {}: {}'.format(
                c1, c2, scipy.stats.pearsonr(datas[c1].Close, datas[c2].Close)))

        # plot heat map of Pearson correlation coefficients
        corr_data = pd.DataFrame({c: v.Close for c, v in datas.items()})
        plt.figure(figsize=(12, 7))
        plt.title('Pearson correlation coefficients between BTC, ETH, XRP, LTC')
        sns.heatmap(corr_data.corr(), vmin=-1.0,
                    vmax=1.0, square=True, annot=True)
        plt.show()


if __name__ == '__main__':
    s = Solution()
    s.quiz1()
