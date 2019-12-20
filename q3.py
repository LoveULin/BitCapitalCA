
def q3(datas: List[int]) -> int:
    '''find the best time point to buy and sell stocks(at most one transaction)'''
    max_profit = 0
    min_price = None
    for item in datas:
        if min_price is None:
            min_price = item
            continue

        if item > min_price:
            if item - min_price > max_profit:
                max_profit = item - min_price
        elif item < min_price:
            min_price = item

    return max_profit


test_list1 = [7, 1, 5, 3, 6, 4]
print(q3(test_list1))

test_list2 = [7, 6, 4, 3, 1]
print(q3(test_list2))
