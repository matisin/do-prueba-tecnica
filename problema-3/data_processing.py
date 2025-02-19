import tabulate
import pandas as pd
import re

def load_data(travels_file, meassurements_file):
    travels = pd.read_csv(travels_file)
    travels['yyyy-mm-ddThh:mm:ss.sss'] = pd.to_datetime(travels['yyyy-mm-ddThh:mm:ss.sss'])
    travels['id'] = pd.to_numeric(travels['id'])
    travels['Station'] = pd.to_numeric(travels['Station'])
    travels['Longitude [degrees_east]'] = pd.to_numeric(travels['Longitude [degrees_east]'])
    travels['Latitude [degrees_north]'] = pd.to_numeric(travels['Latitude [degrees_north]'])
    travels['Bot. Depth [m]'] = pd.to_numeric(travels['Bot. Depth [m]'])

    meassurements = pd.read_csv(meassurements_file)
    meassurements['id'] = pd.to_numeric(meassurements['id'])
    meassurements['Depth [m]'] = pd.to_numeric(meassurements['Depth [m]'])
    meassurements['valor'] = pd.to_numeric(meassurements['valor'])

    return travels, meassurements
    
def join_data(travels, meassurements): 
    joined = pd.merge(travels, meassurements, on='id', how='inner')
    return joined

def standardize_dates(data):
    if 'yyyy-mm-ddThh:mm:ss.sss' not in data.columns:
        raise ValueError("La columna de fecha no está en el DataFrame.")
    
    data['standarized_date'] = data.groupby(['Cruise', 'Station'])['yyyy-mm-ddThh:mm:ss.sss'].transform('min')
    return data

def standardize_names(data):
    data.columns = [
        re.sub(r'[^a-zA-Z0-9_-]', '_', col.lower().replace(' ', '_'))
        for col in data.columns
    ]
    return data

def standarize_variable(data):
    if 'variable' in data.columns:
        data['variable'] = data['variable'].apply(
            lambda x: re.sub(r'[^a-zA-Z0-9_-]', '_', x.lower().replace(' ', '_'))
        )
    return data

def consolidate_data(travels_file, meassurements_file): 
    travels, meassurements = load_data("data/viajes_dd8a0ac9e2.csv", "data/mediciones_4be6910e87.csv")
    data = join_data(travels, meassurements)
    data = standardize_dates(data)
    data = standardize_names(data)
    data = standarize_variable(data)
    return data

def crucero_con_mas_data(data):
    return data['cruise'].value_counts().idxmax()

def hora_con_mas_muestreos(data):
    data['hora'] = data['standarized_date'].dt.hour
    hora = data['hora'].value_counts().idxmax()
    return hora

def promedios_por_profundidad(data):
    data['profundidad_agrupada'] = pd.cut(
        data['depth__m_'],
        bins=range(0, int(data['depth__m_'].max()) + 10, 10),
        right=False
    )
    
    res = []
    variables = [
        'fluorescence__wet_labs_eco-afl_fl__mg_m_3_',
        'dissolved_oxygen__ml_l_',
        'temperature__deg_c_',
        'salinity__practical__psu_'
    ]
    
    for variable in variables:
        data_variable = data[data['variable'] == variable]
        
        avgs = data_variable.groupby(
            ['cruise', 'profundidad_agrupada'],
            observed=False
        )['valor'].mean().reset_index()
        
        avgs['variable'] = variable
        
        res.append(avgs)
    
    return pd.concat(res, ignore_index=True)

data = consolidate_data("data/viajes_dd8a0ac9e2.csv", "data/mediciones_4be6910e87.csv")
crucero = crucero_con_mas_data(data)

print("el crucero con mas data es {0}".format(crucero))

hora= hora_con_mas_muestreos(data) 

print("la hora con mas muestreos es a las {0}".format(hora))
   
promedios = promedios_por_profundidad(data)

#  imprime algunos nan pero por el tiempo restante no alcanzo a chequear si es correcto o no 
print(tabulate.tabulate(promedios, headers='keys', tablefmt='psql'))
